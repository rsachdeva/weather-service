package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rsachdeva/weather-service/internal/forecast"
	"github.com/rsachdeva/weather-service/internal/respond"
)

type WeatherHandler struct {
	nwsClient forecast.Forecaster
	logger    *slog.Logger
}

func NewWeatherHandler(nwsClient forecast.Forecaster, logger *slog.Logger) *WeatherHandler {
	return &WeatherHandler{
		nwsClient: nwsClient,
		logger:    logger,
	}
}

func (wh *WeatherHandler) HandleGetWeather(w http.ResponseWriter, r *http.Request) {
	logger := wh.logger.With("request_id", middleware.GetReqID(r.Context()))
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if latStr == "" || lonStr == "" {
		respond.Error(w, r, http.StatusBadRequest, "lat and lon query parameters are required", logger)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		respond.Error(w, r, http.StatusBadRequest, "lat must be a valid number", logger)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		respond.Error(w, r, http.StatusBadRequest, "lon must be a valid number", logger)
		return
	}

	if lat < -90 || lat > 90 {
		respond.Error(w, r, http.StatusBadRequest, "lat must be between -90 and 90", logger)
		return
	}

	if lon < -180 || lon > 180 {
		respond.Error(w, r, http.StatusBadRequest, "lon must be between -180 and 180", logger)
		return
	}

	fc, err := wh.nwsClient.GetForecast(r.Context(), lat, lon, middleware.GetReqID(r.Context()))
	if errors.Is(err, forecast.ErrPointNotFound) {
		respond.Error(w, r, http.StatusNotFound, "no NWS forecast available for this location", logger)
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "nws forecast failed", "lat", lat, "lon", lon, "err", err)
		respond.Error(w, r, http.StatusInternalServerError, "failed to retrieve forecast", logger)
		return
	}

	// additional fields available from NWS periods[0] if needed:
	// "temperature":      fc.Temperature,
	// "temperature_unit": fc.TemperatureUnit,
	// "wind_speed":       fc.WindSpeed,
	// "wind_direction":   fc.WindDirection,
	respond.JSON(w, r, http.StatusOK, respond.Envelope{
		"forecast":                   fc.ShortForecast,
		"temperature_classification": fc.TemperatureClassification,
	}, logger)
}
