package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"sim-madrasah/backend/internal/auth"
	"sim-madrasah/backend/internal/httpx"
)

type ctxKey string

const ClaimsKey ctxKey = "claims"

const CookieName = "sim_token"

// jeda minimum antar-penulisan terakhir_aktif untuk satu sesi. Tanpa ini setiap
// klik menulis ke database; dengan ini paling banyak sekali per menit per sesi,
// dan itu sudah cukup akurat untuk menampilkan "sedang aktif".
const jedaTandaAktif = time.Minute

type penandaAktif struct {
	mu     sync.Mutex
	terkini map[string]time.Time
}

func (p *penandaAktif) perluTulis(sesiID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if t, ada := p.terkini[sesiID]; ada && now.Sub(t) < jedaTandaAktif {
		return false
	}
	p.terkini[sesiID] = now
	return true
}

// RequireAuth memverifikasi JWT dari cookie (atau header Authorization Bearer),
// LALU memastikan sesinya masih hidup.
//
// Pemeriksaan sesi inilah yang membuat logout benar-benar berarti: tanpa itu
// token tetap sah sampai kedaluwarsa, sehingga menonaktifkan akun tidak
// menendang orangnya keluar dan superadmin tidak bisa memutus sesi siapa pun.
func RequireAuth(secret string, db *sql.DB) func(http.Handler) http.Handler {
	tanda := &penandaAktif{terkini: map[string]time.Time{}}

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
			// Token lama (terbit sebelum pelacakan sesi ada) tidak punya sid.
			// Diperlakukan kedaluwarsa supaya tidak ada token yang lolos tanpa
			// bisa dicabut.
			if claims.SesiID == "" {
				httpx.Error(w, http.StatusUnauthorized, "SESI_LAMA", "Silakan login ulang")
				return
			}

			var berakhir sql.NullString
			var aktif bool
			err = db.QueryRow(`
				SELECT s.berakhir_at, u.is_active
				FROM sesi_login s JOIN users u ON u.id = s.user_id
				WHERE s.id = ? AND s.user_id = ?`, claims.SesiID, claims.UserID).Scan(&berakhir, &aktif)
			switch {
			case err == sql.ErrNoRows:
				httpx.Error(w, http.StatusUnauthorized, "SESI_TIDAK_ADA", "Sesi sudah tidak berlaku")
				return
			case err != nil:
				// Database bermasalah bukan salah pengguna, tapi membiarkannya
				// lewat berarti pemeriksaan sesi bisa dilewati begitu saja.
				log.Printf("middleware: gagal memeriksa sesi %s: %v", claims.SesiID, err)
				httpx.Error(w, http.StatusServiceUnavailable, "DB_ERROR", "Tidak bisa memverifikasi sesi")
				return
			case berakhir.Valid:
				httpx.Error(w, http.StatusUnauthorized, "SESI_BERAKHIR", "Sesi Anda sudah diakhiri. Silakan login lagi.")
				return
			case !aktif:
				httpx.Error(w, http.StatusForbidden, "USER_INACTIVE", "Akun dinonaktifkan")
				return
			}

			if tanda.perluTulis(claims.SesiID) {
				if _, err := db.Exec(
					`UPDATE sesi_login SET terakhir_aktif = NOW() WHERE id = ?`, claims.SesiID); err != nil {
					log.Printf("middleware: gagal memperbarui terakhir_aktif: %v", err)
				}
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
