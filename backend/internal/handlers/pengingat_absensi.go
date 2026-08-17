package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"sim-madrasah/backend/internal/httpx"
)

// StatusKelasAbsensi = kondisi pengisian absensi satu kelas pada satu tanggal.
type StatusKelasAbsensi struct {
	KelasID int64  `json:"kelas_id"`
	Kelas   string `json:"kelas"`
	Santri  int    `json:"santri"`
	Terisi  int    `json:"terisi"`
	// "belum" = belum ada satu pun, "sebagian" = ada tapi belum semua, "lengkap" = semua terisi
	Status string `json:"status"`
}

// StatusAbsensiHarian memeriksa kelas mana saja yang belum/baru sebagian diabsen
// pada tanggal tertentu. Dipakai oleh banner Dashboard maupun worker pengingat,
// supaya keduanya memakai satu logika yang sama.
func StatusAbsensiHarian(db *sql.DB, tanggal string) ([]StatusKelasAbsensi, error) {
	rows, err := db.Query(`
		SELECT k.id, k.nama,
		       COUNT(DISTINCT s.id) AS santri,
		       COUNT(DISTINCT a.santri_id) AS terisi
		FROM kelas k
		JOIN santri s ON s.kelas_id = k.id AND s.is_active = 1
		LEFT JOIN absensi a ON a.santri_id = s.id AND a.tanggal = ?
		WHERE k.aktif = 1
		GROUP BY k.id, k.nama
		ORDER BY k.nama`, tanggal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []StatusKelasAbsensi{}
	for rows.Next() {
		var s StatusKelasAbsensi
		if err := rows.Scan(&s.KelasID, &s.Kelas, &s.Santri, &s.Terisi); err != nil {
			continue
		}
		switch {
		case s.Terisi == 0:
			s.Status = "belum"
		case s.Terisi < s.Santri:
			s.Status = "sebagian"
		default:
			s.Status = "lengkap"
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// HariSekolah: madrasah masuk Sabtu–Rabu (Kamis & Jumat libur) dan mengecualikan
// hari libur yang didaftarkan admin. Konvensi sama dengan dashboard & rekap.
func HariSekolah(db *sql.DB, t time.Time) bool {
	if t.Weekday() == time.Thursday || t.Weekday() == time.Friday {
		return false
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM hari_libur WHERE tanggal = ?`, t.Format("2006-01-02")).Scan(&n)
	return n == 0
}

type pengaturanPengingat struct {
	Aktif bool `json:"aktif"`
	Jam   int  `json:"jam"`
	Menit int  `json:"menit"`
}

func bacaPengaturanPengingat(db *sql.DB) pengaturanPengingat {
	p := pengaturanPengingat{Aktif: true, Jam: 19, Menit: 0}
	_ = db.QueryRow(`SELECT aktif, jam, menit FROM pengingat_absensi_pengaturan WHERE id = 1`).
		Scan(&p.Aktif, &p.Jam, &p.Menit)
	return p
}

// GET /pengingat-absensi?tanggal=YYYY-MM-DD
// Mengembalikan pengaturan + kondisi pengisian absensi hari itu.
func (h *Handler) GetPengingatAbsensi(w http.ResponseWriter, r *http.Request) {
	tglStr := r.URL.Query().Get("tanggal")
	t := time.Now()
	if tglStr != "" {
		if p, err := time.Parse("2006-01-02", tglStr); err == nil {
			t = p
		}
	}
	tanggal := t.Format("2006-01-02")

	daftar, err := StatusAbsensiHarian(h.DB, tanggal)
	if err != nil {
		dbErr(w, err)
		return
	}
	belum, sebagian, lengkap := 0, 0, 0
	for _, k := range daftar {
		switch k.Status {
		case "belum":
			belum++
		case "sebagian":
			sebagian++
		default:
			lengkap++
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"tanggal":      tanggal,
		"hari_sekolah": HariSekolah(h.DB, t),
		"pengaturan":   bacaPengaturanPengingat(h.DB),
		"kelas":        daftar,
		"ringkasan":    map[string]int{"belum": belum, "sebagian": sebagian, "lengkap": lengkap},
	})
}

// POST /pengingat-absensi — ubah nyala/mati & jam pengingat.
func (h *Handler) SetPengingatAbsensi(w http.ResponseWriter, r *http.Request) {
	var req pengaturanPengingat
	if !decode(w, r, &req) {
		return
	}
	if req.Jam < 0 || req.Jam > 23 || req.Menit < 0 || req.Menit > 59 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Jam harus 0–23 dan menit 0–59")
		return
	}
	if _, err := h.DB.Exec(`
		INSERT INTO pengingat_absensi_pengaturan (id, aktif, jam, menit) VALUES (1, ?, ?, ?)
		ON DUPLICATE KEY UPDATE aktif = VALUES(aktif), jam = VALUES(jam), menit = VALUES(menit)`,
		req.Aktif, req.Jam, req.Menit); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, req)
}
