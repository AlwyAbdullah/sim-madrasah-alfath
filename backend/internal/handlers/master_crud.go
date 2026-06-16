package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"

	"sim-madrasah/backend/internal/httpx"
)

// ---- helper error MySQL ----
func dbErr(w http.ResponseWriter, err error) {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		switch me.Number {
		case 1062:
			httpx.Error(w, http.StatusConflict, "DUPLICATE", "Data sudah ada (duplikat).")
			return
		case 1451: // baris masih dirujuk tabel lain
			httpx.Error(w, http.StatusConflict, "IN_USE", "Data masih dipakai dan tidak dapat dihapus.")
			return
		case 1452: // FK tidak ditemukan
			httpx.Error(w, http.StatusBadRequest, "FK_INVALID", "Referensi data tidak valid.")
			return
		}
	}
	httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
}

func decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return false
	}
	return true
}

// ================= KELAS =================
type kelasReq struct {
	Nama    string `json:"nama"`
	Tingkat string `json:"tingkat"`
}

func (h *Handler) CreateKelas(w http.ResponseWriter, r *http.Request) {
	var req kelasReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama kelas wajib diisi")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO kelas (nama, tingkat) VALUES (?, ?)`, req.Nama, nullStr(req.Tingkat))
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdateKelas(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req kelasReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama kelas wajib diisi")
		return
	}
	if _, err := h.DB.Exec(`UPDATE kelas SET nama = ?, tingkat = ? WHERE id = ?`, req.Nama, nullStr(req.Tingkat), id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) DeleteKelas(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM kelas WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// ================= SANTRI =================
type santriReq struct {
	NIS          string `json:"nis"`
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
	KelasID      int64  `json:"kelas_id"`
}

func (h *Handler) CreateSantri(w http.ResponseWriter, r *http.Request) {
	var req santriReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" || (req.JenisKelamin != "L" && req.JenisKelamin != "P") || req.KelasID == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama, jenis kelamin (L/P), dan kelas wajib")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO santri (nis, nama, jenis_kelamin, kelas_id) VALUES (?, ?, ?, ?)`,
		nullStr(req.NIS), req.Nama, req.JenisKelamin, req.KelasID)
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdateSantri(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req santriReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" || (req.JenisKelamin != "L" && req.JenisKelamin != "P") || req.KelasID == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama, jenis kelamin (L/P), dan kelas wajib")
		return
	}
	if _, err := h.DB.Exec(`UPDATE santri SET nis = ?, nama = ?, jenis_kelamin = ?, kelas_id = ? WHERE id = ?`,
		nullStr(req.NIS), req.Nama, req.JenisKelamin, req.KelasID, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// DeleteSantri: soft-delete (is_active=0) agar riwayat absensi/nilai tetap aman.
func (h *Handler) DeleteSantri(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`UPDATE santri SET is_active = 0 WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// ================= MATA PELAJARAN =================
type mapelReq struct {
	Kode  string `json:"kode"`
	Nama  string `json:"nama"`
	Kitab string `json:"kitab"`
}

func (h *Handler) CreateMapel(w http.ResponseWriter, r *http.Request) {
	var req mapelReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama mata pelajaran wajib")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO mata_pelajaran (kode, nama, kitab) VALUES (?, ?, ?)`, nullStr(req.Kode), req.Nama, nullStr(req.Kitab))
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdateMapel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req mapelReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama mata pelajaran wajib")
		return
	}
	if _, err := h.DB.Exec(`UPDATE mata_pelajaran SET kode = ?, nama = ?, kitab = ? WHERE id = ?`, nullStr(req.Kode), req.Nama, nullStr(req.Kitab), id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) DeleteMapel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM mata_pelajaran WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// ================= PERIODE =================
type periodeReq struct {
	Nama        string `json:"nama"`
	TahunAjaran string `json:"tahun_ajaran"`
	Semester    string `json:"semester"`
	IsActive    bool   `json:"is_active"`
}

func (h *Handler) CreatePeriode(w http.ResponseWriter, r *http.Request) {
	var req periodeReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" || req.TahunAjaran == "" || (req.Semester != "ganjil" && req.Semester != "genap") {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama, tahun ajaran, semester (ganjil/genap) wajib")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO periode (nama, tahun_ajaran, semester, is_active) VALUES (?, ?, ?, ?)`,
		req.Nama, req.TahunAjaran, req.Semester, req.IsActive)
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdatePeriode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req periodeReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" || req.TahunAjaran == "" || (req.Semester != "ganjil" && req.Semester != "genap") {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama, tahun ajaran, semester (ganjil/genap) wajib")
		return
	}
	if _, err := h.DB.Exec(`UPDATE periode SET nama = ?, tahun_ajaran = ?, semester = ?, is_active = ? WHERE id = ?`,
		req.Nama, req.TahunAjaran, req.Semester, req.IsActive, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) DeletePeriode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`DELETE FROM periode WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
