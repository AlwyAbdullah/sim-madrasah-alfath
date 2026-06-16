package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
)

// GET /dashboard/summary?kelas_id=&tanggal=
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	tanggal := r.URL.Query().Get("tanggal")
	if tanggal == "" {
		tanggal = time.Now().Format("2006-01-02")
	}

	// total santri (filter kelas opsional)
	totalQuery := `SELECT COUNT(*) FROM santri WHERE is_active = 1`
	absQuery := `
		SELECT a.status, COUNT(*) FROM absensi a
		JOIN santri s ON s.id = a.santri_id AND s.is_active = 1
		WHERE a.tanggal = ?`
	totalArgs := []interface{}{}
	absArgs := []interface{}{tanggal}
	if kelasID != "" {
		totalQuery += ` AND kelas_id = ?`
		totalArgs = append(totalArgs, kelasID)
		absQuery += ` AND s.kelas_id = ?`
		absArgs = append(absArgs, kelasID)
	}
	absQuery += ` GROUP BY a.status`

	var totalSantri int
	_ = h.DB.QueryRow(totalQuery, totalArgs...).Scan(&totalSantri)

	counts := map[string]int{"hadir": 0, "izin": 0, "sakit": 0, "alpha": 0}
	rows, err := h.DB.Query(absQuery, absArgs...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var c int
		_ = rows.Scan(&st, &c)
		counts[st] = c
	}

	persen := 0.0
	if totalSantri > 0 {
		persen = float64(counts["hadir"]) / float64(totalSantri) * 100
		persen = float64(int(persen*10+0.5)) / 10
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"tanggal":              tanggal,
		"total_santri":         totalSantri,
		"hadir":                counts["hadir"],
		"izin":                 counts["izin"],
		"sakit":                counts["sakit"],
		"alpha":                counts["alpha"],
		"persentase_kehadiran": persen,
	})
}

// GET /santri/{id}/detail?periode_id=
func (h *Handler) SantriDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "id tidak valid")
		return
	}
	periodeID := r.URL.Query().Get("periode_id")

	// identitas
	var nis, nama, jk, kelasNama string
	err = h.DB.QueryRow(`
		SELECT COALESCE(s.nis,''), s.nama, s.jenis_kelamin, k.nama
		FROM santri s JOIN kelas k ON k.id = s.kelas_id WHERE s.id = ?`, id).
		Scan(&nis, &nama, &jk, &kelasNama)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Santri tidak ditemukan")
		return
	}

	// rekap kehadiran
	rekap := map[string]int{"hadir": 0, "izin": 0, "sakit": 0, "alpha": 0}
	rows, _ := h.DB.Query(`SELECT status, COUNT(*) FROM absensi WHERE santri_id = ? GROUP BY status`, id)
	if rows != nil {
		for rows.Next() {
			var st string
			var c int
			_ = rows.Scan(&st, &c)
			rekap[st] = c
		}
		rows.Close()
	}

	// nilai per mapel (filter periode opsional)
	nilaiQuery := `
		SELECT mp.nama, n.tugas, n.uts, n.uas, n.nilai_akhir
		FROM nilai n JOIN mata_pelajaran mp ON mp.id = n.mata_pelajaran_id
		WHERE n.santri_id = ?`
	nilaiArgs := []interface{}{id}
	if periodeID != "" {
		nilaiQuery += ` AND n.periode_id = ?`
		nilaiArgs = append(nilaiArgs, periodeID)
	}
	nilaiQuery += ` ORDER BY mp.nama`

	type nilaiRow struct {
		Mapel      string   `json:"mata_pelajaran"`
		Tugas      *float64 `json:"tugas"`
		UTS        *float64 `json:"uts"`
		UAS        *float64 `json:"uas"`
		NilaiAkhir *float64 `json:"nilai_akhir"`
	}
	nilaiList := []nilaiRow{}
	nrows, _ := h.DB.Query(nilaiQuery, nilaiArgs...)
	if nrows != nil {
		for nrows.Next() {
			var nr nilaiRow
			_ = nrows.Scan(&nr.Mapel, &nr.Tugas, &nr.UTS, &nr.UAS, &nr.NilaiAkhir)
			nilaiList = append(nilaiList, nr)
		}
		nrows.Close()
	}

	// rincian ketidakhadiran: tanggal, status, alasan (selain hadir), terbaru dulu
	type absenRow struct {
		Tanggal    string  `json:"tanggal"`
		Status     string  `json:"status"`
		Keterangan *string `json:"keterangan"`
	}
	ketidakhadiran := []absenRow{}
	arows, _ := h.DB.Query(`
		SELECT DATE_FORMAT(tanggal, '%Y-%m-%d'), status, keterangan
		FROM absensi
		WHERE santri_id = ? AND status <> 'hadir'
		ORDER BY tanggal DESC`, id)
	if arows != nil {
		for arows.Next() {
			var ar absenRow
			_ = arows.Scan(&ar.Tanggal, &ar.Status, &ar.Keterangan)
			ketidakhadiran = append(ketidakhadiran, ar)
		}
		arows.Close()
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"santri": map[string]interface{}{
			"id": id, "nis": nis, "nama": nama, "jenis_kelamin": jk, "kelas": kelasNama,
		},
		"kehadiran":      rekap,
		"ketidakhadiran": ketidakhadiran,
		"nilai":          nilaiList,
	})
}
