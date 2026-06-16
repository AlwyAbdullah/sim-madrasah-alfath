package handlers

import (
	"net/http"
	"strings"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/models"
)

func (h *Handler) ListKelas(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, nama, COALESCE(tingkat,'') FROM kelas ORDER BY nama`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	type kelas struct {
		ID      int64  `json:"id"`
		Nama    string `json:"nama"`
		Tingkat string `json:"tingkat"`
	}
	out := []kelas{}
	for rows.Next() {
		var k kelas
		_ = rows.Scan(&k.ID, &k.Nama, &k.Tingkat)
		out = append(out, k)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) ListMapel(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, COALESCE(kode,''), nama, COALESCE(kitab,'') FROM mata_pelajaran ORDER BY nama`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	type mapel struct {
		ID    int64  `json:"id"`
		Kode  string `json:"kode"`
		Nama  string `json:"nama"`
		Kitab string `json:"kitab"`
	}
	out := []mapel{}
	for rows.Next() {
		var m mapel
		_ = rows.Scan(&m.ID, &m.Kode, &m.Nama, &m.Kitab)
		out = append(out, m)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) ListPeriode(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, nama, tahun_ajaran, semester, is_active FROM periode ORDER BY is_active DESC, tahun_ajaran DESC`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	type periode struct {
		ID          int64  `json:"id"`
		Nama        string `json:"nama"`
		TahunAjaran string `json:"tahun_ajaran"`
		Semester    string `json:"semester"`
		IsActive    bool   `json:"is_active"`
	}
	out := []periode{}
	for rows.Next() {
		var p periode
		_ = rows.Scan(&p.ID, &p.Nama, &p.TahunAjaran, &p.Semester, &p.IsActive)
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ListSantri: opsional filter ?kelas_id= dan pencarian ?q=
func (h *Handler) ListSantri(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	query := `SELECT s.id, COALESCE(s.nis,''), s.nama, s.jenis_kelamin, s.kelas_id, k.nama
	          FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE s.is_active = 1`
	args := []interface{}{}
	if kelasID != "" {
		query += ` AND s.kelas_id = ?`
		args = append(args, kelasID)
	}
	if q != "" {
		query += ` AND (s.nama LIKE ? OR s.nis LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	query += ` ORDER BY s.nama`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	out := []models.Santri{}
	for rows.Next() {
		var s models.Santri
		_ = rows.Scan(&s.ID, &s.NIS, &s.Nama, &s.JenisKelamin, &s.KelasID, &s.KelasNama)
		out = append(out, s)
	}
	httpx.JSON(w, http.StatusOK, out)
}
