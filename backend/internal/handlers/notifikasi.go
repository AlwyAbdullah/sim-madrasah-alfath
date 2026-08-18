package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
)

var bulanIndo = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// GET /notifikasi?status=&limit=
func (h *Handler) ListNotifikasi(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	q := `SELECT n.id, COALESCE(s.nama,'-'), n.jenis, n.ref_tanggal, n.tujuan, n.pesan,
	             n.status, n.percobaan, n.catatan, n.dikirim_at, n.created_at
	      FROM notifikasi n LEFT JOIN santri s ON s.id = n.santri_id`
	args := []interface{}{}
	if status != "" {
		q += ` WHERE n.status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY n.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := h.DB.Query(q, args...)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	type item struct {
		ID         int64   `json:"id"`
		Nama       string  `json:"nama"`
		Jenis      string  `json:"jenis"`
		RefTanggal *string `json:"ref_tanggal"`
		Tujuan     string  `json:"tujuan"`
		Pesan      string  `json:"pesan"`
		Status     string  `json:"status"`
		Percobaan  int     `json:"percobaan"`
		Catatan    *string `json:"catatan"`
		DikirimAt  *string `json:"dikirim_at"`
		CreatedAt  string  `json:"created_at"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var ref, kirim sql.NullString
		var created string
		_ = rows.Scan(&it.ID, &it.Nama, &it.Jenis, &ref, &it.Tujuan, &it.Pesan,
			&it.Status, &it.Percobaan, &it.Catatan, &kirim, &created)
		if ref.Valid {
			t := ref.String
			if len(t) > 10 {
				t = t[:10]
			}
			it.RefTanggal = &t
		}
		if kirim.Valid {
			it.DikirimAt = &kirim.String
		}
		it.CreatedAt = created
		out = append(out, it)
	}

	// ringkasan jumlah per status
	ringkas := map[string]int{}
	srows, err := h.DB.Query(`SELECT status, COUNT(*) FROM notifikasi GROUP BY status`)
	if err == nil {
		for srows.Next() {
			var s string
			var c int
			_ = srows.Scan(&s, &c)
			ringkas[s] = c
		}
		srows.Close()
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out, "ringkasan": ringkas})
}

// POST /notifikasi/{id}/ulang — kembalikan ke antrean (untuk yang gagal/batal).
func (h *Handler) UlangNotifikasi(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`UPDATE notifikasi SET status='pending', catatan=NULL WHERE id=?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// POST /notifikasi/{id}/batal — batalkan pesan yang belum terkirim.
func (h *Handler) BatalNotifikasi(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`UPDATE notifikasi SET status='batal' WHERE id=? AND status <> 'terkirim'`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// GET /notifikasi/pengaturan — status aktif/nonaktif worker pengirim.
func (h *Handler) GetPengaturanNotifikasi(w http.ResponseWriter, r *http.Request) {
	var aktif bool
	if err := h.DB.QueryRow(`SELECT aktif FROM notifikasi_pengaturan WHERE id = 1`).Scan(&aktif); err != nil {
		aktif = true // baris belum ada (migrasi tertinggal) -> anggap aktif
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"aktif": aktif})
}

type pengaturanNotifReq struct {
	Aktif bool `json:"aktif"`
}

// POST /notifikasi/pengaturan — nyalakan/matikan pengiriman.
// Saat nonaktif, pesan tetap diantrekan seperti biasa; worker (internal/notifworker)
// hanya berhenti memanggil Telegram sampai diaktifkan lagi.
func (h *Handler) SetPengaturanNotifikasi(w http.ResponseWriter, r *http.Request) {
	var req pengaturanNotifReq
	if !decode(w, r, &req) {
		return
	}
	if _, err := h.DB.Exec(`
		INSERT INTO notifikasi_pengaturan (id, aktif) VALUES (1, ?)
		ON DUPLICATE KEY UPDATE aktif = VALUES(aktif)`, req.Aktif); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"aktif": req.Aktif})
}
