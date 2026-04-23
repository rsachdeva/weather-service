package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Envelope map[string]any

func JSON(w http.ResponseWriter, r *http.Request, status int, data Envelope, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.ErrorContext(r.Context(), "encode error", "err", err)
	}
}

func Error(w http.ResponseWriter, r *http.Request, status int, message string, logger *slog.Logger) {
	logger.ErrorContext(r.Context(), "request error", "status", status, "error", message)
	JSON(w, r, status, Envelope{"error": message}, logger)
}
