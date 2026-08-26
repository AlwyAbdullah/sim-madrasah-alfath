package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/httpx"
)

// waliItem = satu wali kelas.
type waliItem struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Nama     string `json:"nama"`
	Urutan   int    `json:"urutan"` // 1 = wali utama
}

// semuaWali mengembalikan wali seluruh kelas sekaligus (kelas_id -> daftar wali),
// supaya daftar kelas tidak memicu satu query per kelas.
func (h *Handler) semuaWali() (map[int64][]waliItem, error) {
	rows, err := h.DB.Query(`
		SELECT kw.kelas_id, u.id, u.username, u.nama, kw.urutan
		FROM kelas_wali kw JOIN users u ON u.id = kw.user_id
		ORDER BY kw.kelas_id, kw.urutan, u.nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64][]waliItem{}
	for rows.Next() {
		var kelasID int64
		var it waliItem
		if err := rows.Scan(&kelasID, &it.UserID, &it.Username, &it.Nama, &it.Urutan); err != nil {
			continue
		}
		it.Nama = strings.TrimSpace(it.Nama)
		out[kelasID] = append(out[kelasID], it)
	}
	return out, rows.Err()
}

// WaliKelas mengembalikan daftar wali satu kelas. Dipakai worker pengingat juga,
// karena itu ekspor dan menerima *sql.DB, bukan metode handler.
func WaliKelas(db *sql.DB, kelasID int64) ([]waliItem, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.nama, kw.urutan
		FROM kelas_wali kw JOIN users u ON u.id = kw.user_id
		WHERE kw.kelas_id = ? AND u.is_active = 1
		ORDER BY kw.urutan, u.nama`, kelasID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []waliItem{}
	for rows.Next() {
		var it waliItem
		if err := rows.Scan(&it.UserID, &it.Username, &it.Nama, &it.Urutan); err != nil {
			continue
		}
		it.Nama = strings.TrimSpace(it.Nama)
		out = append(out, it)
	}
	return out, rows.Err()
}

// GET /kelas/{id}/wali
func (h *Handler) GetWaliKelas(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var kelasID int64
	if err := h.DB.QueryRow(`SELECT id FROM kelas WHERE id = ?`, id).Scan(&kelasID); err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Kelas tidak ditemukan")
		return
	}
	daftar, err := WaliKelas(h.DB, kelasID)
	if err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, daftar)
}

type setWaliReq struct {
	// Urutan dalam array menentukan urutan wali (yang pertama = wali utama).
	UserIDs []int64 `json:"user_ids"`
}

// PUT /kelas/{id}/wali — ganti seluruh daftar wali kelas.
// Mengirim array kosong berarti kelas ini sengaja tidak punya wali; pengingat
// absensinya tetap dikirim ke grup supaya tidak ada kelas yang luput.
func (h *Handler) SetWaliKelas(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req setWaliReq
	if !decode(w, r, &req) {
		return
	}
	var kelasID int64
	if err := h.DB.QueryRow(`SELECT id FROM kelas WHERE id = ?`, id).Scan(&kelasID); err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Kelas tidak ditemukan")
		return
	}

	// tolak user yang tidak ada / nonaktif sebelum menghapus apa pun
	for _, uid := range req.UserIDs {
		var aktif bool
		if err := h.DB.QueryRow(`SELECT is_active FROM users WHERE id = ?`, uid).Scan(&aktif); err != nil {
			httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Ada akun wali yang tidak ditemukan")
			return
		}
		if !aktif {
			httpx.Error(w, http.StatusBadRequest, "AKUN_NONAKTIF",
				"Akun yang dinonaktifkan tidak bisa dijadikan wali kelas")
			return
		}
	}

	tx, err := h.DB.Begin()
	if err != nil {
		dbErr(w, err)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM kelas_wali WHERE kelas_id = ?`, kelasID); err != nil {
		dbErr(w, err)
		return
	}
	seen := map[int64]bool{}
	urutan := 1
	for _, uid := range req.UserIDs {
		if seen[uid] {
			continue // kirim dua kali orang yang sama -> cukup sekali
		}
		seen[uid] = true
		if _, err := tx.Exec(`
			INSERT INTO kelas_wali (kelas_id, user_id, urutan) VALUES (?, ?, ?)`,
			kelasID, uid, urutan); err != nil {
			dbErr(w, err)
			return
		}
		urutan++
	}
	if err := tx.Commit(); err != nil {
		dbErr(w, err)
		return
	}

	daftar, _ := WaliKelas(h.DB, kelasID)

	var namaKelas string
	_ = h.DB.QueryRow(`SELECT nama FROM kelas WHERE id = ?`, kelasID).Scan(&namaKelas)
	nama := make([]string, 0, len(daftar))
	for _, d := range daftar {
		nama = append(nama, d.Nama)
	}
	ket := strings.Join(nama, ", ")
	if ket == "" {
		ket = "dikosongkan"
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.UbahWaliKelas, Entitas: "kelas", EntitasID: id,
		Ringkasan: fmt.Sprintf("Menetapkan wali %s: %s", namaKelas, ket),
	})

	httpx.JSON(w, http.StatusOK, daftar)
}
