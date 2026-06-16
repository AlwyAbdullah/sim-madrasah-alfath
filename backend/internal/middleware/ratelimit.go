package middleware

import (
	"net/http"
	"sync"
	"time"

	"sim-madrasah/backend/internal/httpx"
)

// RateLimit sederhana berbasis IP (in-memory) — cukup untuk MVP/login.
// Untuk produksi multi-instance gunakan Redis.
type limiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	max     int
	window  time.Duration
}

func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	l := &limiter{hits: make(map[string][]time.Time), max: max, window: window}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()
			l.mu.Lock()
			recent := []time.Time{}
			for _, t := range l.hits[ip] {
				if now.Sub(t) < l.window {
					recent = append(recent, t)
				}
			}
			if len(recent) >= l.max {
				l.mu.Unlock()
				httpx.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak percobaan. Coba lagi nanti.")
				return
			}
			recent = append(recent, now)
			l.hits[ip] = recent
			l.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}
