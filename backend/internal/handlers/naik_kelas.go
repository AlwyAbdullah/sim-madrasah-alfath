package handlers

import (
	"fmt"
	"net/http"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/httpx"
)

type naikKelasReq struct {
	FromKelasID int64 `json:"from_kelas_id"`
	ToKelasID   int64 `json:"to_kelas_id"`
}

// POST /santri/naik-kelas — pindahkan semua santri AKTIF dari kelas asal ke kelas tujuan.
// Hanya mengubah kelas_id (kelas santri saat ini). Nilai/absensi/rapor periode lama
// tetap utuh karena disimpan per periode + riwayat kelas (santri_kelas) sudah beku.
func (h *Handler) NaikKelas(w http.ResponseWriter, r *http.Request) {
	var req naikKelasReq
	if !decode(w, r, &req) {
		return
	}
	if req.FromKelasID == 0 || req.ToKelasID == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Kelas asal dan tujuan wajib")
		return
	}
	if req.FromKelasID == req.ToKelasID {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Kelas asal dan tujuan tidak boleh sama")
		return
	}
	var cnt int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM kelas WHERE id = ?`, req.ToKelasID).Scan(&cnt); err != nil || cnt == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Kelas tujuan tidak ditemukan")
		return
	}
	res, err := h.DB.Exec(`
		UPDATE santri SET kelas_id = ?, updated_by = ?
		WHERE kelas_id = ? AND is_active = 1`, req.ToKelasID, pelaku(r), req.FromKelasID)
	if err != nil {
		dbErr(w, err)
		return
	}
	moved, _ := res.RowsAffected()

	// Perpindahan massal wajib tercatat: satu klik memindahkan puluhan santri,
	// dan tanpa catatan tidak akan terlacak siapa yang melakukannya.
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.NaikKelas, Entitas: "santri",
		Ringkasan: fmt.Sprintf("Memindahkan %d santri dari %s ke %s",
			moved, h.namaKelas(req.FromKelasID), h.namaKelas(req.ToKelasID)),
		Rincian: map[string]interface{}{
			"dari": req.FromKelasID, "ke": req.ToKelasID, "jumlah": moved,
		},
	})

	httpx.JSON(w, http.StatusOK, map[string]interface{}{"moved": moved})
}
