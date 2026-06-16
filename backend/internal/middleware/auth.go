package middleware

import (
	"context"
	"net/http"

	"sim-madrasah/backend/internal/auth"
	"sim-madrasah/backend/internal/httpx"
)

type ctxKey string

const ClaimsKey ctxKey = "claims"

const CookieName = "sim_token"

// RequireAuth memverifikasi JWT dari cookie (atau header Authorization Bearer).
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			if c, err := r.Cookie(CookieName); err == nil {
				tokenStr = c.Value
			} else if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
				tokenStr = h[7:]
			}
			if tokenStr == "" {
				httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
				return
			}
			claims, err := auth.ParseToken(secret, tokenStr)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Sesi tidak valid atau kedaluwarsa")
				return
			}
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFrom(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(ClaimsKey).(*auth.Claims)
	return c
}
