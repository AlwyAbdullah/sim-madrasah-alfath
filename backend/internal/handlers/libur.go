package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
)

func (h *Handler) ListLibur(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, DATE_FORMAT(tanggal,'%Y-%m-%d'), COALESCE(keterangan,'') FROM hari_libur ORDER BY tanggal DESC`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	type libur struct {
		ID         int64  `json:"id"`
		Tanggal    string `json:"tanggal"`
		Keterangan string `json:"keterangan"`
	}
	out := []libur{}
	for rows.Next() {
		var l libur
		_ = rows.Scan(&l.ID, &l.Tanggal, &l.Keterangan)
		out = append(out, l)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type liburReq struct {
	Tanggal    string `json:"tanggal"`
	Keterangan string `json:"keterangan"`
}

func (h *Handler) CreateLibur(w http.ResponseWriter, r *http.Request) {
	var req liburReq
	if !decode(w, r, &req) {
		return
	}
	if req.Tanggal == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Tanggal wajib (YYYY-MM-DD)")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO hari_libur (tanggal, keterangan) VALUES (?, ?)`, req.Tanggal, nullStr(req.Keterangan))
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) DeleteLibur(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM hari_libur WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}
