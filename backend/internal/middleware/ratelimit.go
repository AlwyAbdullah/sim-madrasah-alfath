package middleware

import (
	"net"
	"net/http"
	"strings"
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

// clientIP menentukan kunci pembatas laju.
//
// Header X-Forwarded-For / X-Real-IP HANYA dipercaya bila sambungannya datang
// dari localhost, yaitu dari nginx di mesin yang sama. Alasannya:
//
//   - nginx memakai $proxy_add_x_forwarded_for, yang MENAMBAHKAN IP asli ke
//     nilai kiriman klien. Kalau seluruh isi header dipakai sebagai kunci,
//     penyerang tinggal mengganti-ganti awalannya untuk mendapat jatah baru
//     tiap permintaan — pembatasnya jadi tidak ada gunanya.
//   - backend juga bisa dihubungi langsung (tanpa lewat nginx), dan dalam
//     keadaan itu header tersebut sepenuhnya karangan klien.
//
// Untuk sambungan dari nginx, dipakai entri TERAKHIR pada X-Forwarded-For —
// itulah yang ditambahkan nginx sendiri dan tidak bisa dipalsukan klien.
func clientIP(r *http.Request) string {
	host := hostSaja(r.RemoteAddr)

	if isLoopback(host) {
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			return xri
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			bagian := strings.Split(xff, ",")
			return strings.TrimSpace(bagian[len(bagian)-1])
		}
	}
	// port dibuang: tanpa ini tiap sambungan TCP baru punya kunci berbeda,
	// sehingga hitungannya selalu mulai dari nol
	return host
}

func hostSaja(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

func isLoopback(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return host == "localhost"
}
