package routes_test

import (
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rsachdeva/weather-service/internal/api"
	"github.com/rsachdeva/weather-service/internal/app"
	"github.com/rsachdeva/weather-service/internal/forecast"
	"github.com/rsachdeva/weather-service/internal/routes"
	"github.com/rsachdeva/weather-service/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestApp(forecaster *mocks.Forecaster) *app.Application {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := api.NewWeatherHandler(forecaster, logger)
	return &app.Application{
		Logger:         logger,
		WeatherHandler: handler,
	}
}

func TestRoutes_WithNewTestServer(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	forecasterMock.On("GetForecast", mock.Anything, 33.019844, -96.698883, mock.Anything).
		Return(&forecast.Forecast{
			ShortForecast:             "Sunny",
			TemperatureClassification: "hot",
		}, nil).Once()

	application := newTestApp(forecasterMock)
	mux := routes.SetupRoutes(application)

	ts := httptest.NewTestServer(t, mux)
	client := ts.Client()

	t.Run("health endpoint", func(t *testing.T) {
		resp, err := client.Get("http://weather.local/health")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "OK\n", string(body))
	})

	t.Run("weather endpoint", func(t *testing.T) {
		resp, err := client.Get("http://weather.local/weather?lat=33.019844&lon=-96.698883")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		var res map[string]string
		require.NoError(t, json.UnmarshalRead(resp.Body, &res))
		require.Equal(t, "Sunny", res["forecast"])
		require.Equal(t, "hot", res["temperature_classification"])
	})

	forecasterMock.AssertExpectations(t)
}

// TestRoutes_WithNewServer is the pre-Go 1.27 way of doing what
// TestRoutes_WithNewTestServer does, kept alongside it as a comparison.
//
// Both serve the real chi stack — request ID, logging, recoverer, rate limiter — with
// forecasterMock standing in for the NWS client, so what is under test is routing and
// middleware rather than forecast logic.
//
// What differs is addressing. A loopback server binds a real port, so requests have to
// be built from ts.URL and the server closed by hand. Go 1.27's NewTestServer serves an
// in-memory network whose client accepts any hostname, so the test can name a readable
// stand-in like http://weather.local/ and cleanup is registered through t.Cleanup.
func TestRoutes_WithNewServer(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	forecasterMock.On("GetForecast", mock.Anything, 33.019844, -96.698883, mock.Anything).
		Return(&forecast.Forecast{
			ShortForecast:             "Sunny",
			TemperatureClassification: "hot",
		}, nil).Once()

	application := newTestApp(forecasterMock)
	mux := routes.SetupRoutes(application)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("health endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "OK\n", string(body))
	})

	t.Run("weather endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/weather?lat=33.019844&lon=-96.698883")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		var res map[string]string
		require.NoError(t, json.UnmarshalRead(resp.Body, &res))
		require.Equal(t, "Sunny", res["forecast"])
		require.Equal(t, "hot", res["temperature_classification"])
	})

	forecasterMock.AssertExpectations(t)
}
