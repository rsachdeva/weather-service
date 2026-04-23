package mocks

import (
	"context"

	"github.com/rsachdeva/weather-service/internal/forecast"
	"github.com/stretchr/testify/mock"
)

type Forecaster struct {
	mock.Mock
}

func (m *Forecaster) GetForecast(ctx context.Context, lat, lon float64, requestID string) (*forecast.Forecast, error) {
	args := m.Called(ctx, lat, lon, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*forecast.Forecast), args.Error(1)
}
