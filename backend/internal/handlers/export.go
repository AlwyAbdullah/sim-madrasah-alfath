package handlers

import (
	"fmt"
	"net/http"

	"github.com/xuri/excelize/v2"

	"sim-madrasah/backend/internal/httpx"
)

// GET /nilai/export?kelas_id=&mata_pelajaran_id=&periode_id=
// Mengekspor nilai satu kelas+mapel+periode ke file Excel (.xlsx).
func (h *Handler) ExportNilai(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	mapelID := r.URL.Query().Get("mata_pelajaran_id")
	periodeID := r.URL.Query().Get("periode_id")
	if kelasID == "" || mapelID == "" || periodeID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id, mata_pelajaran_id, periode_id wajib")
		return
	}

	// metadata untuk judul & nama file
	var kelasNama, mapelNama, periodeNama string
	_ = h.DB.QueryRow(`SELECT nama FROM kelas WHERE id = ?`, kelasID).Scan(&kelasNama)
	_ = h.DB.QueryRow(`SELECT nama FROM mata_pelajaran WHERE id = ?`, mapelID).Scan(&mapelNama)
	_ = h.DB.QueryRow(`SELECT nama FROM periode WHERE id = ?`, periodeID).Scan(&periodeNama)

	rows, err := h.DB.Query(`
		SELECT COALESCE(s.nis,''), s.nama, n.tugas, n.uts, n.uas, n.nilai_akhir
		FROM santri s
		LEFT JOIN nilai n ON n.santri_id = s.id AND n.mata_pelajaran_id = ? AND n.periode_id = ?
		WHERE s.kelas_id = ? AND s.is_active = 1
		ORDER BY s.nama`, mapelID, periodeID, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Nilai"
	idx, _ := f.NewSheet(sheet)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")

	// judul
	f.SetCellValue(sheet, "A1", "DAFTAR NILAI")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("Kelas: %s   |   Mata Pelajaran: %s   |   Periode: %s", kelasNama, mapelNama, periodeNama))
	f.SetCellValue(sheet, "A3", "Bobot: Tugas 30% + UTS 30% + UAS 40%")
	f.MergeCell(sheet, "A1", "F1")
	f.MergeCell(sheet, "A2", "F2")
	f.MergeCell(sheet, "A3", "F3")

	// header tabel
	headerRow := 5
	headers := []string{"No", "NIS", "Nama", "Tugas", "UTS", "UAS"}
	headers = append(headers, "Nilai Akhir")
	cols := []string{"A", "B", "C", "D", "E", "F", "G"}
	for i, hd := range headers {
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], headerRow), hd)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", headerRow), fmt.Sprintf("G%d", headerRow), style)

	rowNum := headerRow + 1
	no := 1
	deref := func(p *float64) interface{} {
		if p == nil {
			return ""
		}
		return *p
	}
	for rows.Next() {
		var nis, nama string
		var tugas, uts, uas, akhir *float64
		_ = rows.Scan(&nis, &nama, &tugas, &uts, &uas, &akhir)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), no)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), nis)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), nama)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), deref(tugas))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), deref(uts))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), deref(uas))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), deref(akhir))
		rowNum++
		no++
	}

	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 14)
	f.SetColWidth(sheet, "C", "C", 28)
	f.SetColWidth(sheet, "D", "G", 12)

	filename := fmt.Sprintf("Nilai_%s_%s.xlsx", sanitize(kelasNama), sanitize(mapelNama))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := f.Write(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
	}
}

func sanitize(s string) string {
	out := []rune{}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "export"
	}
	return string(out)
}
