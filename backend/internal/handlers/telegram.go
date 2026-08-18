package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/httpx"
)

// GET /telegram/pengaturan — status token & tujuan chat.
// Token TIDAK pernah dikirim ke browser; hanya statusnya (sudah diisi / belum).
func (h *Handler) GetPengaturanTelegram(w http.ResponseWriter, r *http.Request) {
	var chatID sql.NullString
	_ = h.DB.QueryRow(`SELECT chat_id FROM telegram_pengaturan WHERE id = 1`).Scan(&chatID)

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"token_terpasang": h.Cfg.TelegramBotToken != "",
		"chat_id":         chatID.String,
	})
}

type telegramPengaturanReq struct {
	ChatID string `json:"chat_id"`
}

// POST /telegram/pengaturan — atur tujuan chat (grup guru atau chat pribadi).
func (h *Handler) SetPengaturanTelegram(w http.ResponseWriter, r *http.Request) {
	var req telegramPengaturanReq
	if !decode(w, r, &req) {
		return
	}
	chat := strings.TrimSpace(req.ChatID)
	if len(chat) > 64 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "chat_id terlalu panjang")
		return
	}
	var nilai interface{}
	if chat != "" {
		nilai = chat
	}
	if _, err := h.DB.Exec(`
		INSERT INTO telegram_pengaturan (id, chat_id) VALUES (1, ?)
		ON DUPLICATE KEY UPDATE chat_id = VALUES(chat_id)`, nilai); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"chat_id": chat})
}

// POST /telegram/uji — kirim pesan percobaan supaya admin tahu sambungannya benar
// SEBELUM mengandalkannya untuk pengingat harian.
func (h *Handler) UjiKirimTelegram(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.TelegramBotToken == "" {
		httpx.Error(w, http.StatusBadRequest, "TOKEN_KOSONG",
			"TELEGRAM_BOT_TOKEN belum diisi di server. Hubungi pengelola sistem.")
		return
	}
	var chatID sql.NullString
	_ = h.DB.QueryRow(`SELECT chat_id FROM telegram_pengaturan WHERE id = 1`).Scan(&chatID)
	if !chatID.Valid || strings.TrimSpace(chatID.String) == "" {
		httpx.Error(w, http.StatusBadRequest, "CHAT_KOSONG", "Tujuan chat Telegram belum diatur")
		return
	}

	pesan := fmt.Sprintf(
		"✅ *Uji coba SIM Madrasah Al Fath*\n\nBila pesan ini muncul, sambungan Telegram sudah berhasil.\n\n_Dikirim %s_",
		time.Now().Format("02-01-2006 15:04"))

	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":                  strings.TrimSpace(chatID.String),
		"text":                     pesan,
		"parse_mode":               "Markdown",
		"disable_web_page_preview": true,
	})
	url := fmt.Sprintf("%s/bot%s/sendMessage",
		strings.TrimRight(h.Cfg.TelegramAPIURL, "/"), h.Cfg.TelegramBotToken)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "TELEGRAM_ERROR", "Tidak bisa menghubungi Telegram: "+err.Error())
		return
	}
	defer resp.Body.Close()
	isi, _ := io.ReadAll(io.LimitReader(resp.Body, 500))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		pesanErr := strings.TrimSpace(strings.ReplaceAll(string(isi), "\n", " "))
		if len(pesanErr) > 250 {
			pesanErr = pesanErr[:250]
		}
		httpx.Error(w, http.StatusBadGateway, "TELEGRAM_ERROR",
			fmt.Sprintf("Telegram menolak (%d): %s", resp.StatusCode, pesanErr))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Pesan uji terkirim"})
}
