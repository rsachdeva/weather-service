package forecast

import (
	"context"
	"errors"
)

var ErrPointNotFound = errors.New("point not found in NWS coverage area")

type Forecast struct {
	ShortForecast             string
	Temperature               int
	TemperatureUnit           string
	WindSpeed                 string
	WindDirection             string
	TemperatureClassification string
}

type Forecaster interface {
	GetForecast(ctx context.Context, lat, lon float64, requestID string) (*Forecast, error)
}
