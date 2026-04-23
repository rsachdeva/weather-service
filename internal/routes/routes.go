package routes

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rsachdeva/weather-service/internal/app"
	"github.com/rsachdeva/weather-service/internal/middleware"
)

func SetupRoutes(application *app.Application) *chi.Mux {
	mux := chi.NewRouter()

	mux.Use(chimw.RequestID)
	mux.Use(chimw.RealIP)
	mux.Use(chimw.RequestLogger(&middleware.SlogFormatter{Logger: application.Logger}))
	mux.Use(chimw.Recoverer)
	mux.Use(middleware.RateLimit(application.Logger))

	mux.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "OK\n")
	})
	mux.Get("/weather", application.WeatherHandler.HandleGetWeather)

	return mux
}
