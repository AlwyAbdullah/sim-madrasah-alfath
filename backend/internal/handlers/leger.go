package handlers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/xuri/excelize/v2"

	"sim-madrasah/backend/internal/httpx"
)

type legerMapel struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Kitab string `json:"kitab,omitempty"`
}

type legerRow struct {
	SantriID  int64              `json:"santri_id"`
	NIS       string             `json:"nis"`
	Nama      string             `json:"nama"`
	Nilai     map[int64]*float64 `json:"nilai"` // mapel_id -> nilai akhir
	RataRata  *float64           `json:"rata_rata"`
	Peringkat int                `json:"peringkat"`
}

// buildLeger menyusun matriks nilai akhir + rata-rata + peringkat untuk satu kelas+periode.
func (h *Handler) buildLeger(kelasID, periodeID string) ([]legerMapel, []legerRow, error) {
	// pelajaran kelas dari pemetaan (kelas_mapel) + kitab
	mapels := []legerMapel{}
	mrows, err := h.DB.Query(`
		SELECT km.mata_pelajaran_id, mp.nama, COALESCE(km.kitab,'')
		FROM kelas_mapel km JOIN mata_pelajaran mp ON mp.id = km.mata_pelajaran_id
		WHERE km.kelas_id = ?
		ORDER BY km.urutan, mp.nama`, kelasID)
	if err != nil {
		return nil, nil, err
	}
	for mrows.Next() {
		var m legerMapel
		_ = mrows.Scan(&m.ID, &m.Nama, &m.Kitab)
		mapels = append(mapels, m)
	}
	mrows.Close()

	// fallback: kalau belum ada pemetaan, ambil dari pelajaran yang sudah ada nilainya
	if len(mapels) == 0 {
		frows, err := h.DB.Query(`
			SELECT DISTINCT mp.id, mp.nama
			FROM nilai n JOIN mata_pelajaran mp ON mp.id = n.mata_pelajaran_id
			JOIN santri s ON s.id = n.santri_id
			WHERE s.kelas_id = ? AND n.periode_id = ?
			ORDER BY mp.nama`, kelasID, periodeID)
		if err != nil {
			return nil, nil, err
		}
		for frows.Next() {
			var m legerMapel
			_ = frows.Scan(&m.ID, &m.Nama)
			mapels = append(mapels, m)
		}
		frows.Close()
	}

	// santri kelas
	srows, err := h.DB.Query(`SELECT id, COALESCE(nis,''), nama FROM santri WHERE kelas_id = ? AND is_active = 1 ORDER BY nama`, kelasID)
	if err != nil {
		return nil, nil, err
	}
	rows := []legerRow{}
	idx := map[int64]int{}
	for srows.Next() {
		var r legerRow
		_ = srows.Scan(&r.SantriID, &r.NIS, &r.Nama)
		r.Nilai = map[int64]*float64{}
		idx[r.SantriID] = len(rows)
		rows = append(rows, r)
	}
	srows.Close()

	// nilai akhir per santri per mapel
	nrows, err := h.DB.Query(`
		SELECT n.santri_id, n.mata_pelajaran_id, n.nilai_akhir
		FROM nilai n JOIN santri s ON s.id = n.santri_id
		WHERE s.kelas_id = ? AND n.periode_id = ?`, kelasID, periodeID)
	if err != nil {
		return nil, nil, err
	}
	for nrows.Next() {
		var sid, mid int64
		var akhir *float64
		_ = nrows.Scan(&sid, &mid, &akhir)
		if i, ok := idx[sid]; ok {
			rows[i].Nilai[mid] = akhir
		}
	}
	nrows.Close()

	// rata-rata (hanya mapel yang ada nilainya)
	for i := range rows {
		var sum float64
		var cnt int
		for _, v := range rows[i].Nilai {
			if v != nil {
				sum += *v
				cnt++
			}
		}
		if cnt > 0 {
			avg := float64(int(sum/float64(cnt)*100+0.5)) / 100
			rows[i].RataRata = &avg
		}
	}

	// peringkat berdasar rata-rata (desc); rata-rata sama = peringkat sama
	sorted := make([]int, len(rows))
	for i := range rows {
		sorted[i] = i
	}
	val := func(p *float64) float64 {
		if p == nil {
			return -1
		}
		return *p
	}
	sort.SliceStable(sorted, func(a, b int) bool {
		return val(rows[sorted[a]].RataRata) > val(rows[sorted[b]].RataRata)
	})
	for pos, ri := range sorted {
		if pos > 0 && val(rows[sorted[pos-1]].RataRata) == val(rows[ri].RataRata) {
			rows[ri].Peringkat = rows[sorted[pos-1]].Peringkat
		} else {
			rows[ri].Peringkat = pos + 1
		}
	}

	return mapels, rows, nil
}

// GET /nilai/leger?kelas_id=&periode_id=
func (h *Handler) LegerNilai(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	periodeID := r.URL.Query().Get("periode_id")
	if kelasID == "" || periodeID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id dan periode_id wajib")
		return
	}
	mapels, rows, err := h.buildLeger(kelasID, periodeID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"mapel": mapels,
		"rows":  rows,
	})
}

// GET /nilai/leger/export?kelas_id=&periode_id=
func (h *Handler) ExportLeger(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	periodeID := r.URL.Query().Get("periode_id")
	if kelasID == "" || periodeID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id dan periode_id wajib")
		return
	}
	mapels, rows, err := h.buildLeger(kelasID, periodeID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	var kelasNama, periodeNama string
	_ = h.DB.QueryRow(`SELECT nama FROM kelas WHERE id = ?`, kelasID).Scan(&kelasNama)
	_ = h.DB.QueryRow(`SELECT nama FROM periode WHERE id = ?`, periodeID).Scan(&periodeNama)

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Leger"
	si, _ := f.NewSheet(sheet)
	f.SetActiveSheet(si)
	f.DeleteSheet("Sheet1")

	f.SetCellValue(sheet, "A1", "LEGER NILAI")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("Kelas: %s   |   Periode: %s", kelasNama, periodeNama))

	headerRow := 4
	setCell := func(c, rownum int, v interface{}) {
		name, _ := excelize.CoordinatesToCellName(c, rownum)
		f.SetCellValue(sheet, name, v)
	}
	setCell(1, headerRow, "No")
	setCell(2, headerRow, "NIS")
	setCell(3, headerRow, "Nama")
	for i, m := range mapels {
		setCell(4+i, headerRow, m.Nama)
	}
	rataCol := 4 + len(mapels)
	rankCol := rataCol + 1
	setCell(rataCol, headerRow, "Rata-rata")
	setCell(rankCol, headerRow, "Peringkat")

	hStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	hs, _ := excelize.CoordinatesToCellName(1, headerRow)
	he, _ := excelize.CoordinatesToCellName(rankCol, headerRow)
	f.SetCellStyle(sheet, hs, he, hStyle)

	deref := func(p *float64) interface{} {
		if p == nil {
			return ""
		}
		return *p
	}
	rownum := headerRow + 1
	for i, row := range rows {
		setCell(1, rownum, i+1)
		setCell(2, rownum, row.NIS)
		setCell(3, rownum, row.Nama)
		for j, m := range mapels {
			setCell(4+j, rownum, deref(row.Nilai[m.ID]))
		}
		setCell(rataCol, rownum, deref(row.RataRata))
		setCell(rankCol, rownum, row.Peringkat)
		rownum++
	}

	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 14)
	f.SetColWidth(sheet, "C", "C", 26)

	filename := fmt.Sprintf("Leger_%s_%s.xlsx", sanitize(kelasNama), sanitize(periodeNama))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := f.Write(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
	}
}
