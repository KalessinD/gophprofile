package middleware

import (
	"net/http"

	"github.com/KalessinD/gophprofile/internal/common"
	"golang.org/x/time/rate"
)

// RateLimit middleware limits the number of requests per second using a token bucket algorithm.
func RateLimit(limiter *rate.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", common.AppJSONContentType)
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error": "rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
