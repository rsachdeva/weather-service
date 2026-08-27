package respond

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
)

type Envelope map[string]any

func JSON(w http.ResponseWriter, r *http.Request, status int, data Envelope, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Deterministic sorts map keys. encoding/json v1 did that implicitly; json/v2
	// leaves map order unspecified unless asked, and a stable body matters for
	// response caching and for anything that diffs payloads.
	if err := json.MarshalWrite(w, data, json.Deterministic(true)); err != nil {
		logger.ErrorContext(r.Context(), "encode error", "err", err)
	}
}

func Error(w http.ResponseWriter, r *http.Request, status int, message string, logger *slog.Logger) {
	logger.ErrorContext(r.Context(), "request error", "status", status, "error", message)
	JSON(w, r, status, Envelope{"error": message}, logger)
}
