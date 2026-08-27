package api_test

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rsachdeva/weather-service/internal/api"
	"github.com/rsachdeva/weather-service/internal/forecast"
	"github.com/rsachdeva/weather-service/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newHandler(f *mocks.Forecaster) *api.WeatherHandler {
	return api.NewWeatherHandler(f, slog.Default())
}

func TestHandleGetWeather_MissingParams(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	h := newHandler(forecasterMock)

	req := httptest.NewRequest(http.MethodGet, "/weather", nil)
	rec := httptest.NewRecorder()
	h.HandleGetWeather(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	forecasterMock.AssertNotCalled(t, "GetForecast")
}

func TestHandleGetWeather_InvalidLat(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	h := newHandler(forecasterMock)

	req := httptest.NewRequest(http.MethodGet, "/weather?lat=abc&lon=-96.698883", nil)
	rec := httptest.NewRecorder()
	h.HandleGetWeather(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	forecasterMock.AssertNotCalled(t, "GetForecast")
}

func TestHandleGetWeather_OutOfRangeLat(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	h := newHandler(forecasterMock)

	req := httptest.NewRequest(http.MethodGet, "/weather?lat=91&lon=-96.698883", nil)
	rec := httptest.NewRecorder()
	h.HandleGetWeather(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	forecasterMock.AssertNotCalled(t, "GetForecast")
}

func TestHandleGetWeather_PointNotFound(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	forecasterMock.On("GetForecast", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, forecast.ErrPointNotFound).Once()

	h := newHandler(forecasterMock)
	req := httptest.NewRequest(http.MethodGet, "/weather?lat=33.019844&lon=-96.698883", nil)
	rec := httptest.NewRecorder()
	h.HandleGetWeather(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	forecasterMock.AssertExpectations(t)
}

func TestHandleGetWeather_NWSError(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	forecasterMock.On("GetForecast", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("nws unavailable")).Once()

	h := newHandler(forecasterMock)
	req := httptest.NewRequest(http.MethodGet, "/weather?lat=33.019844&lon=-96.698883", nil)
	rec := httptest.NewRecorder()
	h.HandleGetWeather(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	forecasterMock.AssertExpectations(t)
}

func TestHandleGetWeather_Success(t *testing.T) {
	forecasterMock := &mocks.Forecaster{}
	forecasterMock.On("GetForecast", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&forecast.Forecast{
			ShortForecast:             "Mostly Cloudy",
			TemperatureClassification: "moderate",
		}, nil).Once()

	h := newHandler(forecasterMock)
	req := httptest.NewRequest(http.MethodGet, "/weather?lat=33.019844&lon=-96.698883", nil)
	rec := httptest.NewRecorder()
	h.HandleGetWeather(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.UnmarshalRead(rec.Body, &body))
	require.Equal(t, "Mostly Cloudy", body["forecast"])
	require.Equal(t, "moderate", body["temperature_classification"])
	forecasterMock.AssertExpectations(t)
}
