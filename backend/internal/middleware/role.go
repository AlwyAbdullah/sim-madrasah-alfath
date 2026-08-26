package middleware

import (
	"net/http"

	"sim-madrasah/backend/internal/httpx"
)

// RequireRole membatasi akses ke role tertentu. Harus dipasang setelah RequireAuth.
//
// `superadmin` SELALU diizinkan: perannya adalah admin plus wewenang tambahan
// (mengelola akun admin, mereset password, memutus sesi). Dengan begini seluruh
// rute yang sudah menulis RequireRole("admin") otomatis berlaku untuk superadmin
// tanpa perlu disebut satu per satu — dan tidak ada rute yang terlewat saat
// rute baru ditambahkan nanti.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles)+1)
	for _, r := range roles {
		allowed[r] = true
	}
	allowed["superadmin"] = true

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
