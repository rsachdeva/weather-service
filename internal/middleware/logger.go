package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type SlogFormatter struct {
	Logger *slog.Logger
}

func (f *SlogFormatter) NewLogEntry(r *http.Request) chimw.LogEntry {
	return &slogEntry{
		logger: f.Logger.With(
			"request_id", chimw.GetReqID(r.Context()),
			"method", r.Method,
			"url", r.URL.String(),
			"remote_addr", r.RemoteAddr,
		),
	}
}

type slogEntry struct {
	logger *slog.Logger
}

func (e *slogEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, _ any) {
	e.logger.Info("request", "status", status, "bytes", bytes, "elapsed_ms", elapsed.Milliseconds())
}

func (e *slogEntry) Panic(v any, stack []byte) {
	e.logger.Error("panic", "err", v, "stack", string(stack))
}
