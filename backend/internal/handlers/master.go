package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/models"
)

func (h *Handler) ListKelas(w http.ResponseWriter, r *http.Request) {
	q := `SELECT id, nama, COALESCE(tingkat,''), aktif FROM kelas`
	if r.URL.Query().Get("aktif") == "1" {
		q += ` WHERE aktif = 1`
	}
	q += ` ORDER BY aktif DESC, nama`
	rows, err := h.DB.Query(q)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	type kelas struct {
		ID      int64  `json:"id"`
		Nama    string `json:"nama"`
		Tingkat string `json:"tingkat"`
		Aktif   bool   `json:"aktif"`
	}
	out := []kelas{}
	for rows.Next() {
		var k kelas
		_ = rows.Scan(&k.ID, &k.Nama, &k.Tingkat, &k.Aktif)
		out = append(out, k)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// GET /kelas/{id}/mapel — daftar pelajaran yang dipetakan ke kelas (+ kitab).
func (h *Handler) ListKelasMapel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := h.DB.Query(`
		SELECT km.mata_pelajaran_id, mp.nama, COALESCE(km.kitab,''), km.urutan
		FROM kelas_mapel km JOIN mata_pelajaran mp ON mp.id = km.mata_pelajaran_id
		WHERE km.kelas_id = ?
		ORDER BY km.urutan, mp.nama`, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	type item struct {
		MataPelajaranID int64  `json:"mata_pelajaran_id"`
		Nama            string `json:"nama"`
		Kitab           string `json:"kitab"`
		Urutan          int    `json:"urutan"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		_ = rows.Scan(&it.MataPelajaranID, &it.Nama, &it.Kitab, &it.Urutan)
		out = append(out, it)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type kelasMapelReq struct {
	Items []struct {
		MataPelajaranID int64  `json:"mata_pelajaran_id"`
		Kitab           string `json:"kitab"`
	} `json:"items"`
}

// PUT /kelas/{id}/mapel — set seluruh pemetaan pelajaran kelas (admin).
func (h *Handler) SetKelasMapel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req kelasMapelReq
	if !decode(w, r, &req) {
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM kelas_mapel WHERE kelas_id = ?`, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	stmt, err := tx.Prepare(`INSERT INTO kelas_mapel (kelas_id, mata_pelajaran_id, kitab, urutan) VALUES (?, ?, ?, ?)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer stmt.Close()
	for i, it := range req.Items {
		if _, err := stmt.Exec(id, it.MataPelajaranID, nullStr(it.Kitab), i); err != nil {
			dbErr(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"saved": len(req.Items)})
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

	query := `SELECT s.id, COALESCE(s.nis,''), s.nama, s.jenis_kelamin, COALESCE(s.no_ortu,''), s.kelas_id, k.nama
	          FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE s.is_active = 1`
	args := []interface{}{}
	if r.URL.Query().Get("aktif") == "1" {
		query += ` AND k.aktif = 1`
	}
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
		_ = rows.Scan(&s.ID, &s.NIS, &s.Nama, &s.JenisKelamin, &s.NoOrtu, &s.KelasID, &s.KelasNama)
		out = append(out, s)
	}
	httpx.JSON(w, http.StatusOK, out)
}
