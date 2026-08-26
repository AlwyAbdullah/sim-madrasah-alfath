package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

// buatSesi mencatat satu sesi login baru dan mengembalikan id-nya, yang lalu
// ditanam ke dalam JWT sebagai klaim `sid`.
func (h *Handler) buatSesi(r *http.Request, userID int64) (string, error) {
	id := newBatchID() // UUID v4, generator yang sama dengan batch SPP
	ua := r.UserAgent()
	if len(ua) > 255 {
		ua = ua[:255]
	}
	var ip interface{}
	if v := middleware.ClientIP(r); v != "" {
		ip = v
	}
	_, err := h.DB.Exec(`
		INSERT INTO sesi_login (id, user_id, ip, user_agent) VALUES (?, ?, ?, ?)`,
		id, userID, ip, nullStr(ua))
	return id, err
}

// akhiriSesi menandai sesi selesai. diputusOleh nil = logout sendiri.
func (h *Handler) akhiriSesi(sesiID string, diputusOleh interface{}) error {
	_, err := h.DB.Exec(`
		UPDATE sesi_login SET berakhir_at = NOW(), diputus_oleh = ?
		WHERE id = ? AND berakhir_at IS NULL`, diputusOleh, sesiID)
	return err
}

// akhiriSemuaSesiUser memutus seluruh sesi milik satu akun.
//
// Dipanggil saat akun dinonaktifkan atau passwordnya direset. Tanpa ini
// tindakan tersebut baru terasa setelah tokennya kedaluwarsa — padahal justru
// saat itulah kita ingin orangnya langsung keluar.
func (h *Handler) akhiriSemuaSesiUser(userID interface{}, diputusOleh interface{}) {
	if _, err := h.DB.Exec(`
		UPDATE sesi_login SET berakhir_at = NOW(), diputus_oleh = ?
		WHERE user_id = ? AND berakhir_at IS NULL`, diputusOleh, userID); err != nil {
		log.Printf("sesi: gagal mengakhiri sesi user %v: %v", userID, err)
	}
}

type sesiOut struct {
	ID            string  `json:"id"`
	UserID        int64   `json:"user_id"`
	Username      string  `json:"username"`
	Nama          string  `json:"nama"`
	Role          string  `json:"role"`
	IP            *string `json:"ip"`
	Perangkat     string  `json:"perangkat"`
	DibuatAt      string  `json:"dibuat_at"`
	TerakhirAktif string  `json:"terakhir_aktif"`
	SesiSayaIni   bool    `json:"sesi_saya_ini"`
}

// perangkatDari menyederhanakan User-Agent jadi sesuatu yang enak dibaca.
// Tidak perlu akurat sempurna — gunanya hanya membantu pemilik akun mengenali
// "ini HP saya" versus "ini bukan saya".
func perangkatDari(ua string) string {
	u := strings.ToLower(ua)
	sistem := "Komputer"
	switch {
	case strings.Contains(u, "android"):
		sistem = "Android"
	case strings.Contains(u, "iphone"), strings.Contains(u, "ipad"):
		sistem = "iPhone/iPad"
	case strings.Contains(u, "windows"):
		sistem = "Windows"
	case strings.Contains(u, "mac os"), strings.Contains(u, "macintosh"):
		sistem = "Mac"
	case strings.Contains(u, "linux"):
		sistem = "Linux"
	}
	peramban := ""
	switch {
	// urutannya penting: Edge & Opera juga menyebut "chrome" di UA-nya
	case strings.Contains(u, "edg/"):
		peramban = "Edge"
	case strings.Contains(u, "opr/"), strings.Contains(u, "opera"):
		peramban = "Opera"
	case strings.Contains(u, "chrome"):
		peramban = "Chrome"
	case strings.Contains(u, "firefox"):
		peramban = "Firefox"
	case strings.Contains(u, "safari"):
		peramban = "Safari"
	}
	if peramban == "" {
		return sistem
	}
	return sistem + " · " + peramban
}

// GET /sesi — daftar sesi yang masih hidup.
func (h *Handler) ListSesi(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT s.id, s.user_id, u.username, u.nama, u.role, s.ip, COALESCE(s.user_agent,''),
		       s.dibuat_at, s.terakhir_aktif
		FROM sesi_login s JOIN users u ON u.id = s.user_id
		WHERE s.berakhir_at IS NULL
		ORDER BY s.terakhir_aktif DESC`)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	sesiSaya := ""
	if c := middleware.ClaimsFrom(r); c != nil {
		sesiSaya = c.SesiID
	}

	out := []sesiOut{}
	for rows.Next() {
		var s sesiOut
		var ip sql.NullString
		var ua string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Username, &s.Nama, &s.Role, &ip, &ua,
			&s.DibuatAt, &s.TerakhirAktif); err != nil {
			continue
		}
		s.Nama = strings.TrimSpace(s.Nama)
		if ip.Valid {
			s.IP = &ip.String
		}
		s.Perangkat = perangkatDari(ua)
		s.SesiSayaIni = s.ID == sesiSaya
		out = append(out, s)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// DELETE /sesi/{id} — putus sesi orang lain (khusus superadmin).
func (h *Handler) PutusSesi(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c := middleware.ClaimsFrom(r)
	if c == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi tidak ditemukan")
		return
	}

	var userID int64
	var nama string
	var berakhir sql.NullString
	err := h.DB.QueryRow(`
		SELECT s.user_id, u.nama, s.berakhir_at
		FROM sesi_login s JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`, id).Scan(&userID, &nama, &berakhir)
	if err == sql.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Sesi tidak ditemukan")
		return
	}
	if err != nil {
		dbErr(w, err)
		return
	}
	if berakhir.Valid {
		httpx.Error(w, http.StatusBadRequest, "SUDAH_BERAKHIR", "Sesi ini sudah berakhir")
		return
	}
	// Memutus sesi sendiri lewat tombol ini akan terasa seperti aplikasi rusak —
	// arahkan ke tombol Logout yang memang untuk itu.
	if id == c.SesiID {
		httpx.Error(w, http.StatusBadRequest, "SESI_SENDIRI",
			"Ini sesi Anda sendiri. Gunakan tombol Logout.")
		return
	}

	if err := h.akhiriSesi(id, c.UserID); err != nil {
		dbErr(w, err)
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.PutusSesi, Entitas: "sesi_login", EntitasID: id,
		Ringkasan: fmt.Sprintf("Memutus sesi %s", strings.TrimSpace(nama)),
	})
	httpx.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Sesi %s diputus.", strings.TrimSpace(nama)),
	})
}
