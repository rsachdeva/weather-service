package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/rsachdeva/weather-service/internal/app"
	"github.com/rsachdeva/weather-service/internal/routes"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "port to run server on")
	flag.Parse()

	application, err := app.NewApplication()
	if err != nil {
		panic(err)
	}

	r := routes.SetupRoutes(application)
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application.Logger.Info("server started", "port", port)

	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			application.Logger.Error("server error", "err", serveErr)
		}
	}()

	<-ctx.Done()
	application.Logger.Info("shutting down server")
	if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
		application.Logger.Error("shutdown error", "err", shutdownErr)
	}
}
