package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
)

type guruRow struct {
	ID               int64   `json:"id"`
	Nama             string  `json:"nama"`
	NoRekening       *string `json:"no_rekening"`
	NamaBank         *string `json:"nama_bank"`
	MengajarPerPekan *int    `json:"mengajar_per_pekan"`
	NoTelepon        *string `json:"no_telepon"`
}

func (h *Handler) ListGuru(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, nama, no_rekening, nama_bank, mengajar_per_pekan, no_telepon FROM guru ORDER BY nama`)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()
	out := []guruRow{}
	for rows.Next() {
		var g guruRow
		_ = rows.Scan(&g.ID, &g.Nama, &g.NoRekening, &g.NamaBank, &g.MengajarPerPekan, &g.NoTelepon)
		out = append(out, g)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type guruReq struct {
	Nama             string `json:"nama"`
	NoRekening       string `json:"no_rekening"`
	NamaBank         string `json:"nama_bank"`
	MengajarPerPekan *int   `json:"mengajar_per_pekan"`
	NoTelepon        string `json:"no_telepon"`
}

func (h *Handler) CreateGuru(w http.ResponseWriter, r *http.Request) {
	var req guruReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama guru wajib diisi")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO guru (nama, no_rekening, nama_bank, mengajar_per_pekan, no_telepon) VALUES (?, ?, ?, ?, ?)`,
		req.Nama, nullStr(req.NoRekening), nullStr(req.NamaBank), req.MengajarPerPekan, nullStr(req.NoTelepon))
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdateGuru(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req guruReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama guru wajib diisi")
		return
	}
	if _, err := h.DB.Exec(`UPDATE guru SET nama = ?, no_rekening = ?, nama_bank = ?, mengajar_per_pekan = ?, no_telepon = ? WHERE id = ?`,
		req.Nama, nullStr(req.NoRekening), nullStr(req.NamaBank), req.MengajarPerPekan, nullStr(req.NoTelepon), id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) DeleteGuru(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM guru WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}
