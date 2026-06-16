package middleware

import (
	"net/http"

	"sim-madrasah/backend/internal/httpx"
)

// RequireRole membatasi akses ke role tertentu. Harus dipasang setelah RequireAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := ClaimsFrom(r)
			if c == nil || !allowed[c.Role] {
				httpx.Error(w, http.StatusForbidden, "FORBIDDEN", "Akses khusus admin")
				return
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}
