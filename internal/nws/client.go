package nws

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/rsachdeva/weather-service/internal/forecast"
)

const userAgent = "(weather-service, rohitsachdeva.plano@proton.me)"

type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

type pointsResponse struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

type forecastResponse struct {
	Properties struct {
		Periods []period `json:"periods"`
	} `json:"properties"`
}

type period struct {
	Temperature     int    `json:"temperature"`
	TemperatureUnit string `json:"temperatureUnit"`
	WindSpeed       string `json:"windSpeed"`
	WindDirection   string `json:"windDirection"`
	ShortForecast   string `json:"shortForecast"`
}

// GetForecast calls NWS in two steps:
// Call 1: /points/{lat},{lon}  → resolves lat/lon to NWS office + grid cell, returns forecast URL
// Call 2: {forecastURL}        → returns periods[]; periods[0] is always the current active period
// NWS returns periods in chronological order from now — no name/date matching needed.
// Production: add two-TTL caching (Call 1: 24h, Call 2: 1h) backed by Redis.
func (c *Client) GetForecast(ctx context.Context, lat, lon float64, requestID string) (*forecast.Forecast, error) {
	logger := c.logger.With("request_id", requestID)
	pointsURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)
	logger.InfoContext(ctx, "nws call 1: resolving grid", "lat", lat, "lon", lon, "url", pointsURL)

	var pts pointsResponse
	if err := c.fetch(ctx, pointsURL, &pts, logger); err != nil {
		return nil, fmt.Errorf("points call: %w", err)
	}
	if pts.Properties.Forecast == "" {
		return nil, forecast.ErrPointNotFound
	}
	logger.InfoContext(ctx, "nws call 1: grid resolved", "forecast_url", pts.Properties.Forecast)

	logger.InfoContext(ctx, "nws call 2: fetching forecast", "url", pts.Properties.Forecast)
	var fc forecastResponse
	if err := c.fetch(ctx, pts.Properties.Forecast, &fc, logger); err != nil {
		return nil, fmt.Errorf("forecast call: %w", err)
	}
	if len(fc.Properties.Periods) == 0 {
		return nil, errors.New("no forecast periods returned by NWS")
	}

	p := fc.Properties.Periods[0]
	logger.InfoContext(ctx, "nws call 2: forecast received",
		"period_name", p.ShortForecast,
		"temperature", p.Temperature,
		"temperature_unit", p.TemperatureUnit,
		"classification", classify(p.Temperature),
	)

	return &forecast.Forecast{
		ShortForecast:             p.ShortForecast,
		Temperature:               p.Temperature,
		TemperatureUnit:           p.TemperatureUnit,
		WindSpeed:                 p.WindSpeed,
		WindDirection:             p.WindDirection,
		TemperatureClassification: classify(p.Temperature),
	}, nil
}

func (c *Client) fetch(ctx context.Context, url string, dst any, logger *slog.Logger) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if logger.Enabled(ctx, slog.LevelDebug) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
		logger.DebugContext(ctx, "nws raw response", "url", url, "status", resp.StatusCode, "body", jsontext.Value(body))
		if resp.StatusCode == http.StatusNotFound {
			return forecast.ErrPointNotFound
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
		}
		if err := json.Unmarshal(body, dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	}

	// INFO path: stream directly, no allocation
	if resp.StatusCode == http.StatusNotFound {
		return forecast.ErrPointNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	if err := json.UnmarshalRead(resp.Body, dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func classify(temp int) string {
	switch {
	case temp >= 100:
		return "very hot"
	case temp >= 85:
		return "hot"
	case temp <= 30:
		return "very cold"
	case temp <= 55:
		return "cold"
	default:
		return "moderate"
	}
}
