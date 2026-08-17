package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"

	"sim-madrasah/backend/internal/httpx"
)

// Hari efektif madrasah = Sabtu–Rabu. DAYOFWEEK MySQL: 5=Kamis, 6=Jumat.
// Hari libur yang didaftarkan admin juga dikecualikan. Konvensi ini sama dengan
// yang dipakai dashboard, supaya angkanya tidak berbeda antar halaman.
const filterHariEfektif = `
	AND DAYOFWEEK(a.tanggal) NOT IN (5,6)
	AND a.tanggal NOT IN (SELECT tanggal FROM hari_libur)`

type rekapAngka struct {
	Hadir  int     `json:"hadir"`
	Izin   int     `json:"izin"`
	Sakit  int     `json:"sakit"`
	Alpha  int     `json:"alpha"`
	Total  int     `json:"total"`
	Persen float64 `json:"persen"`
}

func (r *rekapAngka) hitung() {
	r.Total = r.Hadir + r.Izin + r.Sakit + r.Alpha
	if r.Total > 0 {
		r.Persen = float64(int(float64(r.Hadir)/float64(r.Total)*1000+0.5)) / 10
	}
}

func (r *rekapAngka) tambah(status string, n int) {
	switch status {
	case "hadir":
		r.Hadir += n
	case "izin":
		r.Izin += n
	case "sakit":
		r.Sakit += n
	case "alpha":
		r.Alpha += n
	}
}

// rentangRekap membaca from/to (YYYY-MM-DD). Default = bulan berjalan.
func rentangRekap(r *http.Request) (string, string, error) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		now := time.Now()
		awal := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		from = awal.Format("2006-01-02")
		to = awal.AddDate(0, 1, -1).Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return "", "", err
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return "", "", err
	}
	return from, to, nil
}

type rekapKelas struct {
	KelasID int64  `json:"kelas_id"`
	Kelas   string `json:"kelas"`
	Santri  int    `json:"santri"`
	rekapAngka
}

type rekapBulan struct {
	Bulan string `json:"bulan"` // YYYY-MM
	rekapAngka
}

// alphaTanggal = satu kejadian alpha beserta keterangan bila guru mengisinya.
type alphaTanggal struct {
	Tanggal    string `json:"tanggal"` // YYYY-MM-DD
	Keterangan string `json:"keterangan,omitempty"`
}

type rekapSantri struct {
	SantriID int64  `json:"santri_id"`
	Nama     string `json:"nama"`
	Kelas    string `json:"kelas"`
	rekapAngka
	// Rincian tanggal alpha agar bisa langsung ditindaklanjuti (dihubungi wali, dll).
	TanggalAlpha []alphaTanggal `json:"tanggal_alpha"`
}

// dataRekap menyusun seluruh angka rekap dalam satu tempat, dipakai oleh
// endpoint JSON maupun ekspor Excel agar keduanya tidak mungkin berbeda.
func (h *Handler) dataRekap(from, to, kelasID string) (rekapAngka, int, []rekapKelas, []rekapBulan, []rekapSantri, error) {
	var total rekapAngka
	var hariEfektif int
	perKelas := []rekapKelas{}
	perBulan := []rekapBulan{}
	perSantri := []rekapSantri{}

	filterKelas := ""
	argsKelas := []interface{}{}
	if kelasID != "" {
		filterKelas = ` AND s.kelas_id = ?`
		argsKelas = append(argsKelas, kelasID)
	}
	args := func(extra ...interface{}) []interface{} {
		out := []interface{}{from, to}
		out = append(out, argsKelas...)
		return append(out, extra...)
	}

	// jumlah hari yang benar-benar ada catatan absensinya (hari efektif nyata)
	if err := h.DB.QueryRow(`
		SELECT COUNT(DISTINCT a.tanggal)
		FROM absensi a JOIN santri s ON s.id = a.santri_id
		WHERE a.tanggal BETWEEN ? AND ?`+filterKelas+filterHariEfektif, args()...).
		Scan(&hariEfektif); err != nil {
		return total, 0, nil, nil, nil, err
	}

	// ringkasan keseluruhan
	rows, err := h.DB.Query(`
		SELECT a.status, COUNT(*)
		FROM absensi a JOIN santri s ON s.id = a.santri_id
		WHERE a.tanggal BETWEEN ? AND ?`+filterKelas+filterHariEfektif+`
		GROUP BY a.status`, args()...)
	if err != nil {
		return total, 0, nil, nil, nil, err
	}
	for rows.Next() {
		var st string
		var n int
		_ = rows.Scan(&st, &n)
		total.tambah(st, n)
	}
	rows.Close()
	total.hitung()

	// per kelas
	rows, err = h.DB.Query(`
		SELECT k.id, k.nama, a.status, COUNT(*), COUNT(DISTINCT s.id)
		FROM absensi a
		JOIN santri s ON s.id = a.santri_id
		JOIN kelas k ON k.id = s.kelas_id
		WHERE a.tanggal BETWEEN ? AND ?`+filterKelas+filterHariEfektif+`
		GROUP BY k.id, k.nama, a.status
		ORDER BY k.nama`, args()...)
	if err != nil {
		return total, 0, nil, nil, nil, err
	}
	idxKelas := map[int64]int{}
	for rows.Next() {
		var id int64
		var nama, st string
		var n, jmlSantri int
		_ = rows.Scan(&id, &nama, &st, &n, &jmlSantri)
		i, ada := idxKelas[id]
		if !ada {
			perKelas = append(perKelas, rekapKelas{KelasID: id, Kelas: nama})
			i = len(perKelas) - 1
			idxKelas[id] = i
		}
		perKelas[i].tambah(st, n)
		if jmlSantri > perKelas[i].Santri {
			perKelas[i].Santri = jmlSantri
		}
	}
	rows.Close()
	for i := range perKelas {
		perKelas[i].hitung()
	}

	// tren per bulan
	rows, err = h.DB.Query(`
		SELECT DATE_FORMAT(a.tanggal, '%Y-%m'), a.status, COUNT(*)
		FROM absensi a JOIN santri s ON s.id = a.santri_id
		WHERE a.tanggal BETWEEN ? AND ?`+filterKelas+filterHariEfektif+`
		GROUP BY 1, a.status
		ORDER BY 1`, args()...)
	if err != nil {
		return total, 0, nil, nil, nil, err
	}
	idxBulan := map[string]int{}
	for rows.Next() {
		var bln, st string
		var n int
		_ = rows.Scan(&bln, &st, &n)
		i, ada := idxBulan[bln]
		if !ada {
			perBulan = append(perBulan, rekapBulan{Bulan: bln})
			i = len(perBulan) - 1
			idxBulan[bln] = i
		}
		perBulan[i].tambah(st, n)
	}
	rows.Close()
	for i := range perBulan {
		perBulan[i].hitung()
	}

	// per santri (untuk daftar "perlu perhatian")
	rows, err = h.DB.Query(`
		SELECT s.id, s.nama, k.nama, a.status, COUNT(*)
		FROM absensi a
		JOIN santri s ON s.id = a.santri_id
		JOIN kelas k ON k.id = s.kelas_id
		WHERE a.tanggal BETWEEN ? AND ?`+filterKelas+filterHariEfektif+`
		GROUP BY s.id, s.nama, k.nama, a.status`, args()...)
	if err != nil {
		return total, 0, nil, nil, nil, err
	}
	idxSantri := map[int64]int{}
	for rows.Next() {
		var id int64
		var nama, kelas, st string
		var n int
		_ = rows.Scan(&id, &nama, &kelas, &st, &n)
		i, ada := idxSantri[id]
		if !ada {
			perSantri = append(perSantri, rekapSantri{SantriID: id, Nama: nama, Kelas: kelas})
			i = len(perSantri) - 1
			idxSantri[id] = i
		}
		perSantri[i].tambah(st, n)
	}
	rows.Close()
	for i := range perSantri {
		perSantri[i].hitung()
	}

	// rincian TANGGAL setiap alpha (beserta keterangan bila ada)
	rows, err = h.DB.Query(`
		SELECT a.santri_id, a.tanggal, COALESCE(a.keterangan, '')
		FROM absensi a JOIN santri s ON s.id = a.santri_id
		WHERE a.status = 'alpha' AND a.tanggal BETWEEN ? AND ?`+filterKelas+filterHariEfektif+`
		ORDER BY a.santri_id, a.tanggal`, args()...)
	if err != nil {
		return total, 0, nil, nil, nil, err
	}
	for rows.Next() {
		var sid int64
		var tgl time.Time
		var ket string
		if err := rows.Scan(&sid, &tgl, &ket); err != nil {
			continue
		}
		if i, ada := idxSantri[sid]; ada {
			perSantri[i].TanggalAlpha = append(perSantri[i].TanggalAlpha,
				alphaTanggal{Tanggal: tgl.Format("2006-01-02"), Keterangan: ket})
		}
	}
	rows.Close()

	// urutkan: alpha terbanyak dulu, lalu persentase kehadiran terendah
	for i := 1; i < len(perSantri); i++ {
		for j := i; j > 0; j-- {
			a, b := perSantri[j-1], perSantri[j]
			if b.Alpha > a.Alpha || (b.Alpha == a.Alpha && b.Persen < a.Persen) {
				perSantri[j-1], perSantri[j] = perSantri[j], perSantri[j-1]
			} else {
				break
			}
		}
	}

	return total, hariEfektif, perKelas, perBulan, perSantri, nil
}

// GET /absensi/rekap?from=&to=&kelas_id=
func (h *Handler) RekapAbsensi(w http.ResponseWriter, r *http.Request) {
	from, to, err := rentangRekap(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
		return
	}
	kelasID := r.URL.Query().Get("kelas_id")

	total, hariEfektif, perKelas, perBulan, perSantri, err := h.dataRekap(from, to, kelasID)
	if err != nil {
		dbErr(w, err)
		return
	}

	// batasi daftar "perlu perhatian" agar respons tetap ringan
	perhatian := perSantri
	if len(perhatian) > 100 {
		perhatian = perhatian[:100]
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"from":         from,
		"to":           to,
		"hari_efektif": hariEfektif,
		"ringkasan":    total,
		"per_kelas":    perKelas,
		"per_bulan":    perBulan,
		"perhatian":    perhatian,
	})
}

// GET /absensi/rekap/export?from=&to=&kelas_id=&label=
func (h *Handler) ExportRekapAbsensi(w http.ResponseWriter, r *http.Request) {
	from, to, err := rentangRekap(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
		return
	}
	kelasID := r.URL.Query().Get("kelas_id")
	label := r.URL.Query().Get("label")
	if label == "" {
		label = from + " s/d " + to
	}

	total, hariEfektif, perKelas, perBulan, perSantri, err := h.dataRekap(from, to, kelasID)
	if err != nil {
		dbErr(w, err)
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	judul, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 13}})
	kepala, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// ---- Sheet 1: Ringkasan + per kelas ----
	s1 := "Ringkasan"
	idx, _ := f.NewSheet(s1)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")

	f.SetCellValue(s1, "A1", "REKAP KEHADIRAN SANTRI")
	f.SetCellStyle(s1, "A1", "A1", judul)
	f.SetCellValue(s1, "A2", "Periode: "+label)
	f.SetCellValue(s1, "A3", fmt.Sprintf("Hari efektif tercatat: %d hari (Sabtu–Rabu, di luar hari libur)", hariEfektif))
	f.SetCellValue(s1, "A5", "Hadir")
	f.SetCellValue(s1, "B5", total.Hadir)
	f.SetCellValue(s1, "A6", "Izin")
	f.SetCellValue(s1, "B6", total.Izin)
	f.SetCellValue(s1, "A7", "Sakit")
	f.SetCellValue(s1, "B7", total.Sakit)
	f.SetCellValue(s1, "A8", "Alpha")
	f.SetCellValue(s1, "B8", total.Alpha)
	f.SetCellValue(s1, "A9", "Persentase kehadiran")
	f.SetCellValue(s1, "B9", fmt.Sprintf("%.1f%%", total.Persen))

	baris := 11
	f.SetCellValue(s1, fmt.Sprintf("A%d", baris), "PER KELAS")
	f.SetCellStyle(s1, fmt.Sprintf("A%d", baris), fmt.Sprintf("A%d", baris), judul)
	baris++
	for i, h := range []string{"Kelas", "Santri", "Hadir", "Izin", "Sakit", "Alpha", "Total", "% Hadir"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, baris)
		f.SetCellValue(s1, cell, h)
	}
	c1, _ := excelize.CoordinatesToCellName(1, baris)
	c2, _ := excelize.CoordinatesToCellName(8, baris)
	f.SetCellStyle(s1, c1, c2, kepala)
	baris++
	for _, k := range perKelas {
		vals := []interface{}{k.Kelas, k.Santri, k.Hadir, k.Izin, k.Sakit, k.Alpha, k.Total, fmt.Sprintf("%.1f%%", k.Persen)}
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, baris)
			f.SetCellValue(s1, cell, v)
		}
		baris++
	}

	baris++
	f.SetCellValue(s1, fmt.Sprintf("A%d", baris), "TREN PER BULAN")
	f.SetCellStyle(s1, fmt.Sprintf("A%d", baris), fmt.Sprintf("A%d", baris), judul)
	baris++
	for i, h := range []string{"Bulan", "Hadir", "Izin", "Sakit", "Alpha", "Total", "% Hadir"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, baris)
		f.SetCellValue(s1, cell, h)
	}
	c1, _ = excelize.CoordinatesToCellName(1, baris)
	c2, _ = excelize.CoordinatesToCellName(7, baris)
	f.SetCellStyle(s1, c1, c2, kepala)
	baris++
	for _, b := range perBulan {
		vals := []interface{}{b.Bulan, b.Hadir, b.Izin, b.Sakit, b.Alpha, b.Total, fmt.Sprintf("%.1f%%", b.Persen)}
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, baris)
			f.SetCellValue(s1, cell, v)
		}
		baris++
	}
	f.SetColWidth(s1, "A", "A", 24)
	f.SetColWidth(s1, "B", "H", 10)

	// ---- Sheet 2: Per santri ----
	s2 := "Per Santri"
	f.NewSheet(s2)
	f.SetCellValue(s2, "A1", "KEHADIRAN PER SANTRI — "+label)
	f.SetCellStyle(s2, "A1", "A1", judul)
	f.SetCellValue(s2, "A2", "Diurutkan dari alpha terbanyak")
	for i, h := range []string{"No", "Nama", "Kelas", "Hadir", "Izin", "Sakit", "Alpha", "Total", "% Hadir", "Tanggal Alpha"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		f.SetCellValue(s2, cell, h)
	}
	c1, _ = excelize.CoordinatesToCellName(1, 4)
	c2, _ = excelize.CoordinatesToCellName(10, 4)
	f.SetCellStyle(s2, c1, c2, kepala)
	for i, s := range perSantri {
		tglAlpha := ""
		for j, t := range s.TanggalAlpha {
			if j > 0 {
				tglAlpha += ", "
			}
			tglAlpha += t.Tanggal
			if t.Keterangan != "" {
				tglAlpha += " (" + t.Keterangan + ")"
			}
		}
		vals := []interface{}{i + 1, s.Nama, s.Kelas, s.Hadir, s.Izin, s.Sakit, s.Alpha, s.Total,
			fmt.Sprintf("%.1f%%", s.Persen), tglAlpha}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+5)
			f.SetCellValue(s2, cell, v)
		}
	}
	f.SetColWidth(s2, "A", "A", 5)
	f.SetColWidth(s2, "B", "B", 28)
	f.SetColWidth(s2, "C", "C", 14)
	f.SetColWidth(s2, "D", "I", 9)
	f.SetColWidth(s2, "J", "J", 60)

	filename := fmt.Sprintf("RekapKehadiran_%s_%s.xlsx", from, to)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := f.Write(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
	}
}
