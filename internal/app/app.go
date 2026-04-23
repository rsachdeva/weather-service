package app

import (
	"log/slog"
	"os"

	"github.com/rsachdeva/weather-service/internal/api"
	"github.com/rsachdeva/weather-service/internal/nws"
)

type Application struct {
	Logger         *slog.Logger
	WeatherHandler *api.WeatherHandler
}

func NewApplication() (*Application, error) {
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	logger.Info("log level configured", "debug_enabled", level == slog.LevelDebug)
	nwsClient := nws.NewClient(logger)
	weatherHandler := api.NewWeatherHandler(nwsClient, logger)

	return &Application{
		Logger:         logger,
		WeatherHandler: weatherHandler,
	}, nil
}
