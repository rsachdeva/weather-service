package nws

import (
	"bytes"
	_ "embed"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rsachdeva/weather-service/internal/forecast"
	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		temp           int
		classification string
	}{
		{105, "very hot"},
		{100, "very hot"},
		{99, "hot"},
		{85, "hot"},
		{84, "moderate"},
		{56, "moderate"},
		{55, "cold"},
		{31, "cold"},
		{30, "very cold"},
		{-10, "very cold"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.classification, classify(tc.temp), "classify(%d)", tc.temp)
	}
}

//go:embed testdata/points.json
var pointsJSON []byte

//go:embed testdata/forecast.json
var forecastJSON []byte

func stubNWSMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	serve := func(body []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/geo+json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}
	}
	mux.Handle("GET /points/33.0198,-96.6989", serve(pointsJSON))
	mux.Handle("GET /gridpoints/FWD/88,104/forecast", serve(forecastJSON))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected request", http.StatusNotFound)
	})
	return mux
}

func TestGetForecast_WithNewTestServer(t *testing.T) {
	ts := httptest.NewTestServer(t, stubNWSMux(t))

	client := &Client{
		httpClient: ts.Client(),
		logger:     slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	fc, err := client.GetForecast(t.Context(), 33.019844, -96.698883, "test-req-newtestserver")
	require.NoError(t, err)
	require.NotNil(t, fc)
	require.Equal(t, "Partly Cloudy", fc.ShortForecast)
	require.Equal(t, 75, fc.Temperature)
	require.Equal(t, "F", fc.TemperatureUnit)
	require.Equal(t, "10 mph", fc.WindSpeed)
	require.Equal(t, "S", fc.WindDirection)
	require.Equal(t, "moderate", fc.TemperatureClassification)
}

type rewriteTransport struct {
	targetURL *url.URL
	base      http.RoundTripper
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())
	reqCopy.URL.Host = r.targetURL.Host
	return r.base.RoundTrip(reqCopy)
}

// TestGetForecast_WithNewTLSServer is the pre-Go 1.27 way of doing what
// TestGetForecast_WithNewTestServer does, kept alongside it as a comparison.
//
// Both aim Client at stubNWSMux, which serves captured api.weather.gov GeoJSON from
// testdata. Those payloads are written as raw bytes rather than marshalled from
// pointsResponse/forecastResponse: encoding with the same structs the client decodes
// into would let a wrong json tag agree with itself and pass. stubNWSMux is a stub,
// not a mock — it verifies no calls, and every assertion is on the decoded
// *forecast.Forecast.
//
// What differs is how the client is aimed at the stub. GetForecast hardcodes
// https://api.weather.gov, and a loopback httptest server only intercepts
// example.com and *.example.com; any other hostname is dialed for real. Hence
// rewriteTransport. Only URL.Host needs rewriting, since NewTLSServer already speaks
// https, and rewriting to 127.0.0.1 lands inside the httptest certificate's SANs, so
// ts.Client()'s transport still verifies the chain rather than skipping it.
//
// Go 1.27's NewTestServer removes that plumbing. It serves both http and https over
// an in-memory network, its client routes every hostname to the server (setting
// InsecureSkipVerify rather than depending on the certificate matching
// api.weather.gov), and it registers t.Cleanup. No listener, no port, no defer Close.
func TestGetForecast_WithNewTLSServer(t *testing.T) {
	ts := httptest.NewTLSServer(stubNWSMux(t))
	defer ts.Close()

	targetURL, err := url.Parse(ts.URL)
	require.NoError(t, err)

	client := &Client{
		httpClient: &http.Client{
			Transport: &rewriteTransport{
				targetURL: targetURL,
				base:      ts.Client().Transport,
			},
		},
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	fc, err := client.GetForecast(t.Context(), 33.019844, -96.698883, "test-req-newtlsserver")
	require.NoError(t, err)
	require.NotNil(t, fc)
	require.Equal(t, "Partly Cloudy", fc.ShortForecast)
	require.Equal(t, 75, fc.Temperature)
	require.Equal(t, "F", fc.TemperatureUnit)
	require.Equal(t, "10 mph", fc.WindSpeed)
	require.Equal(t, "S", fc.WindDirection)
	require.Equal(t, "moderate", fc.TemperatureClassification)
}

const (
	pointsPath   = "/points/33.0198,-96.6989"
	forecastPath = "/gridpoints/FWD/88,104/forecast"
)

func newStubbedClient(t *testing.T, h http.Handler, level slog.Level) (*Client, *bytes.Buffer) {
	t.Helper()
	ts := httptest.NewTestServer(t, h)
	var logs bytes.Buffer
	return &Client{
		httpClient: ts.Client(),
		logger:     slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: level})),
	}, &logs
}

func servePoints(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == pointsPath {
			_, _ = w.Write(pointsJSON)
			return
		}
		next(w, r)
	}
}

func TestGetForecast_Errors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantIs  error
		wantMsg string
	}{
		{
			name:    "points call returns 404",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
			wantIs:  forecast.ErrPointNotFound,
		},
		{
			name:    "points call returns 500",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantMsg: "unexpected status 500",
		},
		{
			name:    "points call returns malformed json",
			handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"properties":`) },
			wantMsg: "decode response",
		},
		{
			name:    "points call omits forecast url",
			handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"properties":{}}`) },
			wantIs:  forecast.ErrPointNotFound,
		},
		{
			name:    "forecast call returns 404",
			handler: servePoints(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }),
			wantIs:  forecast.ErrPointNotFound,
		},
		{
			name:    "forecast call returns 500",
			handler: servePoints(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }),
			wantMsg: "unexpected status 500",
		},
		{
			name: "forecast call returns no periods",
			handler: servePoints(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"properties":{"periods":[]}}`)
			}),
			wantMsg: "no forecast periods",
		},
	}

	// fetch duplicates its status and decode handling across a debug branch that buffers
	// the body and an info branch that streams it, so every case runs against both.
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
		t.Run(level.String(), func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					client, _ := newStubbedClient(t, tc.handler, level)

					fc, err := client.GetForecast(t.Context(), 33.019844, -96.698883, "test-req-errors")
					require.Error(t, err)
					require.Nil(t, fc)
					if tc.wantIs != nil {
						require.ErrorIs(t, err, tc.wantIs)
					}
					if tc.wantMsg != "" {
						require.ErrorContains(t, err, tc.wantMsg)
					}
				})
			}
		})
	}
}

func TestGetForecast_DebugLogsRawBody(t *testing.T) {
	client, logs := newStubbedClient(t, stubNWSMux(t), slog.LevelDebug)

	fc, err := client.GetForecast(t.Context(), 33.019844, -96.698883, "test-req-debug")
	require.NoError(t, err)
	require.Equal(t, "Partly Cloudy", fc.ShortForecast)
	require.Equal(t, "moderate", fc.TemperatureClassification)

	require.Contains(t, logs.String(), "nws raw response")
	require.Contains(t, logs.String(), "test-req-debug")
	require.Contains(t, logs.String(), forecastPath)
}
