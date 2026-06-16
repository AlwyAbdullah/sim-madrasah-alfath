package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

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
