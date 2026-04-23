package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

const (
	ratePerSecond = 5
	burst         = 10
)

type ipRateLimiter struct {
	limiters sync.Map
}

func (l *ipRateLimiter) get(ip string) *rate.Limiter {
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip
	}
	v, _ := l.limiters.LoadOrStore(host, rate.NewLimiter(ratePerSecond, burst))
	return v.(*rate.Limiter)
}

func RateLimit(logger *slog.Logger) func(http.Handler) http.Handler {
	l := &ipRateLimiter{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.get(r.RemoteAddr).Allow() {
				logger.WarnContext(r.Context(), "rate limit exceeded", "remote_addr", r.RemoteAddr)
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
