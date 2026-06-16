package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
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

var bulanID = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// rentang tanggal untuk filter absensi
func rentangAbsensi(rng string, now time.Time) (string, string) {
	end := now.Format("2006-01-02")
	y, m := now.Year(), int(now.Month())
	switch rng {
	case "mingguan":
		return now.AddDate(0, 0, -6).Format("2006-01-02"), end
	case "bulanan":
		return fmt.Sprintf("%04d-%02d-01", y, m), end
	case "semester":
		if m >= 7 { // Ganjil Jul–Des
			return fmt.Sprintf("%04d-07-01", y), end
		}
		return fmt.Sprintf("%04d-01-01", y), end // Genap Jan–Jun
	default: // tahun = tahun ajaran berjalan (Jul–Jun)
		sy := y
		if m < 7 {
			sy = y - 1
		}
		return fmt.Sprintf("%04d-07-01", sy), end
	}
}

// GET /santri/{id}/detail?periode_id=&range=mingguan|bulanan|semester|tahun
func (h *Handler) SantriDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "id tidak valid")
		return
	}
	periodeID := r.URL.Query().Get("periode_id")
	rng := r.URL.Query().Get("range")
	if rng == "" {
		rng = "tahun"
	}
	start, end := rentangAbsensi(rng, time.Now())

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

	// rekap kehadiran dalam rentang — kecualikan Kamis(5)/Jumat(6) & hari libur
	rekap := map[string]int{"hadir": 0, "izin": 0, "sakit": 0, "alpha": 0}
	rows, _ := h.DB.Query(`
		SELECT status, COUNT(*) FROM absensi
		WHERE santri_id = ? AND tanggal BETWEEN ? AND ?
		  AND DAYOFWEEK(tanggal) NOT IN (5,6)
		  AND tanggal NOT IN (SELECT tanggal FROM hari_libur)
		GROUP BY status`, id, start, end)
	if rows != nil {
		for rows.Next() {
			var st string
			var c int
			_ = rows.Scan(&st, &c)
			rekap[st] = c
		}
		rows.Close()
	}
	totalEfektif := rekap["hadir"] + rekap["izin"] + rekap["sakit"] + rekap["alpha"]
	persen := 0.0
	if totalEfektif > 0 {
		persen = float64(int(float64(rekap["hadir"])/float64(totalEfektif)*1000+0.5)) / 10
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
		  AND tanggal BETWEEN ? AND ?
		  AND DAYOFWEEK(tanggal) NOT IN (5,6)
		  AND tanggal NOT IN (SELECT tanggal FROM hari_libur)
		ORDER BY tanggal DESC`, id, start, end)
	if arows != nil {
		for arows.Next() {
			var ar absenRow
			_ = arows.Scan(&ar.Tanggal, &ar.Status, &ar.Keterangan)
			ketidakhadiran = append(ketidakhadiran, ar)
		}
		arows.Close()
	}

	resp := map[string]interface{}{
		"santri": map[string]interface{}{
			"id": id, "nis": nis, "nama": nama, "jenis_kelamin": jk, "kelas": kelasNama,
		},
		"range":                rng,
		"kehadiran":            rekap,
		"persentase_kehadiran": persen,
		"ketidakhadiran":       ketidakhadiran,
		"nilai":                nilaiList,
	}

	// SPP terlambat — KHUSUS ADMIN
	if c := middleware.ClaimsFrom(r); c != nil && c.Role == "admin" {
		resp["spp_terlambat"] = h.sppTerlambat(id, time.Now())
	}

	httpx.JSON(w, http.StatusOK, resp)
}

type sppLate struct {
	Tahun int    `json:"tahun"`
	Bulan int    `json:"bulan"`
	Label string `json:"label"`
}

// sppTerlambat: bulan SPP (tahun ajaran Jul–Jun berjalan) yang sudah jatuh tempo tapi belum lunas.
func (h *Handler) sppTerlambat(santriID int64, now time.Time) []sppLate {
	y, m := now.Year(), int(now.Month())
	startYear := y
	if m < 7 {
		startYear = y - 1
	}
	// 12 bulan TA: Jul..Des (startYear), Jan..Jun (startYear+1)
	type ym struct{ Y, M int }
	bulanTA := []ym{}
	for mm := 7; mm <= 12; mm++ {
		bulanTA = append(bulanTA, ym{startYear, mm})
	}
	for mm := 1; mm <= 6; mm++ {
		bulanTA = append(bulanTA, ym{startYear + 1, mm})
	}
	// status lunas
	paid := map[string]bool{}
	prows, _ := h.DB.Query(`SELECT tahun, bulan FROM spp WHERE santri_id = ? AND lunas = 1`, santriID)
	if prows != nil {
		for prows.Next() {
			var ty, tb int
			_ = prows.Scan(&ty, &tb)
			paid[fmt.Sprintf("%d-%d", ty, tb)] = true
		}
		prows.Close()
	}
	late := []sppLate{}
	for _, b := range bulanTA {
		jatuhTempo := b.Y < y || (b.Y == y && b.M <= m) // sudah lewat/berjalan
		if jatuhTempo && !paid[fmt.Sprintf("%d-%d", b.Y, b.M)] {
			late = append(late, sppLate{Tahun: b.Y, Bulan: b.M, Label: fmt.Sprintf("%s %d", bulanID[b.M-1], b.Y)})
		}
	}
	return late
}
