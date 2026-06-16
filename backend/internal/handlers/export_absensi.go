package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"

	"sim-madrasah/backend/internal/httpx"
)

var statusInisial = map[string]string{"hadir": "H", "izin": "I", "sakit": "S", "alpha": "A"}

// GET /absensi/export?kelas_id=&bulan=YYYY-MM
// Rekap absensi bulanan: kolom per tanggal + total H/I/S/A per santri.
func (h *Handler) ExportAbsensi(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	bulan := r.URL.Query().Get("bulan")
	if kelasID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id wajib")
		return
	}
	if bulan == "" {
		bulan = time.Now().Format("2006-01")
	}
	first, err := time.Parse("2006-01", bulan)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format bulan harus YYYY-MM")
		return
	}
	last := first.AddDate(0, 1, -1)
	jmlHari := last.Day()
	start := first.Format("2006-01-02")
	end := last.Format("2006-01-02")

	var kelasNama string
	_ = h.DB.QueryRow(`SELECT nama FROM kelas WHERE id = ?`, kelasID).Scan(&kelasNama)

	// daftar santri kelas
	type santri struct {
		id   int64
		nis  string
		nama string
	}
	srows, err := h.DB.Query(`SELECT id, COALESCE(nis,''), nama FROM santri WHERE kelas_id = ? AND is_active = 1 ORDER BY nama`, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var list []santri
	for srows.Next() {
		var s santri
		_ = srows.Scan(&s.id, &s.nis, &s.nama)
		list = append(list, s)
	}
	srows.Close()

	// absensi bulan ini → map[santri_id][day]status
	data := map[int64]map[int]string{}
	arows, err := h.DB.Query(`
		SELECT a.santri_id, a.tanggal, a.status
		FROM absensi a JOIN santri s ON s.id = a.santri_id
		WHERE s.kelas_id = ? AND a.tanggal BETWEEN ? AND ?`, kelasID, start, end)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	for arows.Next() {
		var sid int64
		var tgl time.Time
		var st string
		if err := arows.Scan(&sid, &tgl, &st); err != nil {
			continue
		}
		if data[sid] == nil {
			data[sid] = map[int]string{}
		}
		data[sid][tgl.Day()] = st
	}
	arows.Close()

	// ---- bangun Excel ----
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Rekap Absensi"
	idx, _ := f.NewSheet(sheet)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")

	f.SetCellValue(sheet, "A1", "REKAP ABSENSI BULANAN")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("Kelas: %s   |   Bulan: %s", kelasNama, first.Format("January 2006")))
	f.SetCellValue(sheet, "A3", "Keterangan: H=Hadir, I=Izin, S=Sakit, A=Alpha")

	headerRow := 5
	// kolom: No, NIS, Nama, 1..jmlHari, H, I, S, A
	col := 1
	setCell := func(c, rownum int, v interface{}) {
		name, _ := excelize.CoordinatesToCellName(c, rownum)
		f.SetCellValue(sheet, name, v)
	}
	setCell(col, headerRow, "No")
	setCell(col+1, headerRow, "NIS")
	setCell(col+2, headerRow, "Nama")
	dayStartCol := col + 3
	for d := 1; d <= jmlHari; d++ {
		setCell(dayStartCol+d-1, headerRow, d)
	}
	totalStartCol := dayStartCol + jmlHari
	setCell(totalStartCol, headerRow, "H")
	setCell(totalStartCol+1, headerRow, "I")
	setCell(totalStartCol+2, headerRow, "S")
	setCell(totalStartCol+3, headerRow, "A")

	// style header
	hStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	hStart, _ := excelize.CoordinatesToCellName(1, headerRow)
	hEnd, _ := excelize.CoordinatesToCellName(totalStartCol+3, headerRow)
	f.SetCellStyle(sheet, hStart, hEnd, hStyle)

	// baris data
	rownum := headerRow + 1
	for i, s := range list {
		setCell(1, rownum, i+1)
		setCell(2, rownum, s.nis)
		setCell(3, rownum, s.nama)
		var cntH, cntI, cntS, cntA int
		for d := 1; d <= jmlHari; d++ {
			if st, ok := data[s.id][d]; ok {
				setCell(dayStartCol+d-1, rownum, statusInisial[st])
				switch st {
				case "hadir":
					cntH++
				case "izin":
					cntI++
				case "sakit":
					cntS++
				case "alpha":
					cntA++
				}
			}
		}
		setCell(totalStartCol, rownum, cntH)
		setCell(totalStartCol+1, rownum, cntI)
		setCell(totalStartCol+2, rownum, cntS)
		setCell(totalStartCol+3, rownum, cntA)
		rownum++
	}

	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 14)
	f.SetColWidth(sheet, "C", "C", 26)

	filename := fmt.Sprintf("RekapAbsensi_%s_%s.xlsx", sanitize(kelasNama), bulan)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := f.Write(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
	}
}
