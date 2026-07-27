package handlers

import (
	"net/http"

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
	res, err := h.DB.Exec(`UPDATE santri SET kelas_id = ? WHERE kelas_id = ? AND is_active = 1`, req.ToKelasID, req.FromKelasID)
	if err != nil {
		dbErr(w, err)
		return
	}
	moved, _ := res.RowsAffected()
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"moved": moved})
}
