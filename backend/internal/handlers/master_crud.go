package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

// pelaku mengembalikan id pengguna untuk kolom updated_by, atau NULL bila tidak
// diketahui (mis. dipanggil di luar konteks permintaan yang terautentikasi).
func pelaku(r *http.Request) interface{} {
	if c := middleware.ClaimsFrom(r); c != nil {
		return c.UserID
	}
	return nil
}

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
	Aktif   *bool  `json:"aktif"`
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
	aktif := true
	if req.Aktif != nil {
		aktif = *req.Aktif
	}
	res, err := h.DB.Exec(`INSERT INTO kelas (nama, tingkat, aktif) VALUES (?, ?, ?)`, req.Nama, nullStr(req.Tingkat), aktif)
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
	aktif := true
	if req.Aktif != nil {
		aktif = *req.Aktif
	}
	if _, err := h.DB.Exec(`UPDATE kelas SET nama = ?, tingkat = ?, aktif = ?, updated_by = ? WHERE id = ?`,
		req.Nama, nullStr(req.Tingkat), aktif, pelaku(r), id); err != nil {
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
	NoOrtu       string `json:"no_ortu"`
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
	res, err := h.DB.Exec(`INSERT INTO santri (nis, nama, jenis_kelamin, no_ortu, kelas_id) VALUES (?, ?, ?, ?, ?)`,
		nullStr(req.NIS), req.Nama, req.JenisKelamin, nullStr(req.NoOrtu), req.KelasID)
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.TambahSantri, Entitas: "santri", EntitasID: strconv.FormatInt(id, 10),
		Ringkasan: fmt.Sprintf("Menambah santri %s ke %s", req.Nama, h.namaKelas(req.KelasID)),
	})
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// namaKelas dipakai untuk menyusun ringkasan log yang bisa dibaca manusia —
// "Kelas 3" jauh lebih berguna daripada "kelas_id 5".
func (h *Handler) namaKelas(id int64) string {
	var nama string
	if err := h.DB.QueryRow(`SELECT nama FROM kelas WHERE id = ?`, id).Scan(&nama); err != nil {
		return "-"
	}
	return nama
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
	if _, err := h.DB.Exec(`
		UPDATE santri SET nis = ?, nama = ?, jenis_kelamin = ?, no_ortu = ?, kelas_id = ?, updated_by = ?
		WHERE id = ?`,
		nullStr(req.NIS), req.Nama, req.JenisKelamin, nullStr(req.NoOrtu), req.KelasID,
		pelaku(r), id); err != nil {
		dbErr(w, err)
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.UbahSantri, Entitas: "santri", EntitasID: id,
		Ringkasan: fmt.Sprintf("Mengubah data santri %s (%s)", req.Nama, h.namaKelas(req.KelasID)),
	})
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// DeleteSantri: soft-delete (is_active=0) agar riwayat absensi/nilai tetap aman.
func (h *Handler) DeleteSantri(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var nama string
	_ = h.DB.QueryRow(`SELECT nama FROM santri WHERE id = ?`, id).Scan(&nama)
	if _, err := h.DB.Exec(`UPDATE santri SET is_active = 0, updated_by = ? WHERE id = ?`,
		pelaku(r), id); err != nil {
		dbErr(w, err)
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.HapusSantri, Entitas: "santri", EntitasID: id,
		Ringkasan: fmt.Sprintf("Menonaktifkan santri %s", nama),
	})
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
	if _, err := h.DB.Exec(`UPDATE mata_pelajaran SET kode = ?, nama = ?, kitab = ?, updated_by = ? WHERE id = ?`,
		nullStr(req.Kode), req.Nama, nullStr(req.Kitab), pelaku(r), id); err != nil {
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
	if _, err := h.DB.Exec(`
		UPDATE periode SET nama = ?, tahun_ajaran = ?, semester = ?, is_active = ?, updated_by = ?
		WHERE id = ?`,
		req.Nama, req.TahunAjaran, req.Semester, req.IsActive, pelaku(r), id); err != nil {
		dbErr(w, err)
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.UbahMaster, Entitas: "periode", EntitasID: id,
		Ringkasan: fmt.Sprintf("Mengubah periode %s (%s %s)", req.Nama, req.TahunAjaran, req.Semester),
	})
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
