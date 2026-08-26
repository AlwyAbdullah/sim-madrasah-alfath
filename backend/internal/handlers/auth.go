package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/auth"
	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
	"sim-madrasah/backend/internal/models"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Username dan password wajib diisi")
		return
	}

	var (
		id           int64
		passwordHash string
		nama, role   string
		isActive     bool
	)
	err := h.DB.QueryRow(
		`SELECT id, password_hash, nama, role, is_active FROM users WHERE username = ?`,
		req.Username,
	).Scan(&id, &passwordHash, &nama, &role, &isActive)

	// Pesan generik untuk mencegah enumerasi user.
	if err == sql.ErrNoRows || !auth.CheckPassword(passwordHash, req.Password) {
		// Percobaan yang GAGAL ikut dicatat: lonjakannya adalah tanda pertama
		// ada yang menebak-nebak password — risiko nyata selama password awal
		// masih seragam.
		audit.CatatUser(h.DB, r, 0, req.Username, req.Username, audit.Entri{
			Aksi:      audit.Login,
			Ringkasan: fmt.Sprintf("Percobaan login GAGAL untuk %q", req.Username),
		})
		httpx.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Username atau password salah")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", "Terjadi kesalahan server")
		return
	}
	if !isActive {
		httpx.Error(w, http.StatusForbidden, "USER_INACTIVE", "Akun dinonaktifkan")
		return
	}

	token, err := auth.GenerateToken(h.Cfg.JWTSecret, h.Cfg.JWTExpiryMin, id, req.Username, role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat sesi")
		return
	}

	// dicatat setelah token berhasil dibuat; kegagalan di sini tidak boleh
	// menggagalkan login yang sudah sah
	if _, err := h.DB.Exec(`UPDATE users SET terakhir_login = NOW() WHERE id = ?`, id); err != nil {
		log.Printf("auth: gagal mencatat terakhir_login untuk user %d: %v", id, err)
	}
	audit.CatatUser(h.DB, r, id, req.Username, nama, audit.Entri{
		Aksi:      audit.Login,
		Ringkasan: fmt.Sprintf("%s masuk ke sistem", nama),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(time.Duration(h.Cfg.JWTExpiryMin) * time.Minute),
	})

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"user": models.User{ID: id, Username: req.Username, Nama: nama, Role: role},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Rute logout sengaja TIDAK di balik RequireAuth: cookie yang sudah
	// kedaluwarsa pun harus tetap bisa dibersihkan. Karena itu claims-nya
	// dibaca sendiri di sini — kalau mengandalkan middleware, pelakunya selalu
	// tidak dikenal dan logout tidak pernah tercatat.
	if c := h.claimsDariCookie(r); c != nil {
		var nama string
		if err := h.DB.QueryRow(`SELECT nama FROM users WHERE id = ?`, c.UserID).Scan(&nama); err != nil || nama == "" {
			nama = c.Username
		}
		audit.CatatUser(h.DB, r, c.UserID, c.Username, nama, audit.Entri{
			Aksi:      audit.Logout,
			Ringkasan: fmt.Sprintf("%s keluar dari sistem", nama),
		})
	}
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Berhasil logout"})
}

// claimsDariCookie membaca JWT dari cookie/header tanpa mewajibkannya sah.
// Mengembalikan nil bila tidak ada atau tidak valid.
func (h *Handler) claimsDariCookie(r *http.Request) *auth.Claims {
	tokenStr := ""
	if c, err := r.Cookie(middleware.CookieName); err == nil {
		tokenStr = c.Value
	} else if hdr := r.Header.Get("Authorization"); len(hdr) > 7 && hdr[:7] == "Bearer " {
		tokenStr = hdr[7:]
	}
	if tokenStr == "" {
		return nil
	}
	c, err := auth.ParseToken(h.Cfg.JWTSecret, tokenStr)
	if err != nil {
		return nil
	}
	return c
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}
	var nama string
	_ = h.DB.QueryRow(`SELECT nama FROM users WHERE id = ?`, c.UserID).Scan(&nama)
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"user": models.User{ID: c.UserID, Username: c.Username, Nama: nama, Role: c.Role},
	})
}
