package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

// masaBerlakuKode: kode penautan sengaja berumur pendek. Kode yang berlaku
// selamanya sama saja dengan password kedua yang tidak pernah kedaluwarsa.
const masaBerlakuKode = 15 * time.Minute

// hurufKode tanpa 0/O/1/I supaya tidak salah baca saat diketik ulang di HP.
const hurufKode = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func kodeAcak(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, v := range b {
		out[i] = hurufKode[int(v)%len(hurufKode)]
	}
	return string(out), nil
}

type tautanOut struct {
	Tertaut      bool   `json:"tertaut"`
	NamaTelegram string `json:"nama_telegram,omitempty"`
	BotUsername  string `json:"bot_username,omitempty"`
	Kode         string `json:"kode,omitempty"`
	Tautan       string `json:"tautan,omitempty"`
	BerlakuSampai string `json:"berlaku_sampai,omitempty"`
	TokenAda     bool   `json:"token_ada"`
}

func (h *Handler) botUsername() string {
	var u sql.NullString
	_ = h.DB.QueryRow(`SELECT bot_username FROM telegram_pengaturan WHERE id = 1`).Scan(&u)
	return strings.TrimSpace(u.String)
}

// GET /telegram/tautan — status penautan akun yang sedang login.
func (h *Handler) StatusTautanTelegram(w http.ResponseWriter, r *http.Request) {
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}
	var tgID sql.NullInt64
	var tgNama sql.NullString
	_ = h.DB.QueryRow(`SELECT telegram_user_id, telegram_nama FROM users WHERE id = ?`, c.UserID).
		Scan(&tgID, &tgNama)

	out := tautanOut{
		Tertaut:     tgID.Valid,
		BotUsername: h.botUsername(),
		TokenAda:    h.Cfg.TelegramBotToken != "",
	}
	if tgNama.Valid {
		out.NamaTelegram = tgNama.String
	}
	httpx.JSON(w, http.StatusOK, out)
}

// POST /telegram/tautan — terbitkan kode sekali pakai untuk akun yang login.
func (h *Handler) BuatKodeTautanTelegram(w http.ResponseWriter, r *http.Request) {
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}
	if h.Cfg.TelegramBotToken == "" {
		httpx.Error(w, http.StatusBadRequest, "TOKEN_KOSONG",
			"Bot Telegram belum dipasang di server. Hubungi pengelola sistem.")
		return
	}

	kode, err := kodeAcak(6)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "KODE_ERROR", "Gagal membuat kode")
		return
	}
	exp := time.Now().Add(masaBerlakuKode)
	if _, err := h.DB.Exec(`
		UPDATE users SET telegram_kode = ?, telegram_kode_exp = ? WHERE id = ?`,
		kode, exp, c.UserID); err != nil {
		dbErr(w, err)
		return
	}

	out := tautanOut{
		BotUsername:   h.botUsername(),
		Kode:          kode,
		BerlakuSampai: exp.Format("15:04"),
		TokenAda:      true,
	}
	if out.BotUsername != "" {
		// Telegram meneruskan bagian setelah ?start= sebagai isi pesan /start,
		// jadi penggunanya cukup menekan tautan lalu tombol START.
		out.Tautan = fmt.Sprintf("https://t.me/%s?start=%s", out.BotUsername, kode)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// DELETE /telegram/tautan — lepaskan penautan akun yang sedang login.
func (h *Handler) HapusTautanTelegram(w http.ResponseWriter, r *http.Request) {
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}
	if _, err := h.DB.Exec(`
		UPDATE users SET telegram_user_id = NULL, telegram_nama = NULL,
		                 telegram_kode = NULL, telegram_kode_exp = NULL
		 WHERE id = ?`, c.UserID); err != nil {
		dbErr(w, err)
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.UbahPengaturan, Entitas: "users",
		Ringkasan: fmt.Sprintf("%s melepas tautan Telegram", c.Username),
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Tautan Telegram dilepas"})
}
