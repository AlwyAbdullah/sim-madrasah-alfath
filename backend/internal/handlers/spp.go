package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

var bulanSingkat = []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// urutan bulan tahun ajaran: Juli .. Juni
var urutanBulanTA = []int{7, 8, 9, 10, 11, 12, 1, 2, 3, 4, 5, 6}

// tahun kalender dari (TA start, bulan): Jul..Des = startYear, Jan..Jun = startYear+1
func calTahun(startYear, bulan int) int {
	if bulan >= 7 {
		return startYear
	}
	return startYear + 1
}

func taStartSekarang(now time.Time) int {
	if int(now.Month()) < 7 {
		return now.Year() - 1
	}
	return now.Year()
}

type sppItem struct {
	SantriID int64        `json:"santri_id"`
	NIS      string       `json:"nis"`
	Nama     string       `json:"nama"`
	Bulan    map[int]bool `json:"bulan"` // key = bulan kalender (1..12) -> lunas
	Lunas    int          `json:"lunas"`
}

// GET /spp?kelas_id=&tahun=  (tahun = tahun ajaran mulai, mis. 2025 = TA 2025/2026)
func (h *Handler) GetSPP(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	tahunStr := r.URL.Query().Get("tahun")
	if kelasID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id wajib")
		return
	}
	startYear := taStartSekarang(time.Now())
	if tahunStr != "" {
		fmt.Sscanf(tahunStr, "%d", &startYear)
	}

	srows, err := h.DB.Query(`SELECT id, COALESCE(nis,''), nama FROM santri WHERE kelas_id = ? AND is_active = 1 ORDER BY nama`, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	items := []sppItem{}
	idx := map[int64]int{}
	for srows.Next() {
		var it sppItem
		_ = srows.Scan(&it.SantriID, &it.NIS, &it.Nama)
		it.Bulan = map[int]bool{}
		idx[it.SantriID] = len(items)
		items = append(items, it)
	}
	srows.Close()

	prows, err := h.DB.Query(`
		SELECT sp.santri_id, sp.bulan, sp.lunas
		FROM spp sp JOIN santri s ON s.id = sp.santri_id
		WHERE s.kelas_id = ? AND ((sp.tahun = ? AND sp.bulan >= 7) OR (sp.tahun = ? AND sp.bulan <= 6))`,
		kelasID, startYear, startYear+1)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	for prows.Next() {
		var sid, bln int64
		var lunas bool
		_ = prows.Scan(&sid, &bln, &lunas)
		if i, ok := idx[sid]; ok && lunas {
			items[i].Bulan[int(bln)] = true
			items[i].Lunas++
		}
	}
	prows.Close()

	httpx.JSON(w, http.StatusOK, map[string]interface{}{"tahun": startYear, "urutan_bulan": urutanBulanTA, "items": items})
}

type sppToggleReq struct {
	SantriID int64 `json:"santri_id"`
	Tahun    int   `json:"tahun"` // TA start year
	Bulan    int   `json:"bulan"` // bulan kalender 1..12
	Lunas    bool  `json:"lunas"`
}

// POST /spp/toggle
func (h *Handler) ToggleSPP(w http.ResponseWriter, r *http.Request) {
	var req sppToggleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return
	}
	if req.SantriID == 0 || req.Tahun == 0 || req.Bulan < 1 || req.Bulan > 12 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "santri_id, tahun, bulan (1-12) wajib")
		return
	}
	calY := calTahun(req.Tahun, req.Bulan)

	claims := middleware.ClaimsFrom(r)
	var userID interface{}
	if claims != nil {
		userID = claims.UserID
	}
	var tgl interface{}
	if req.Lunas {
		tgl = time.Now().Format("2006-01-02")
	}

	_, err := h.DB.Exec(`
		INSERT INTO spp (santri_id, tahun, bulan, lunas, tanggal_bayar, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE lunas = VALUES(lunas), tanggal_bayar = VALUES(tanggal_bayar)`,
		req.SantriID, calY, req.Bulan, req.Lunas, tgl, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"santri_id": req.SantriID, "bulan": req.Bulan, "lunas": req.Lunas})
}

// GET /spp/export?kelas_id=&tahun=
func (h *Handler) ExportSPP(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	tahunStr := r.URL.Query().Get("tahun")
	if kelasID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id wajib")
		return
	}
	startYear := taStartSekarang(time.Now())
	if tahunStr != "" {
		fmt.Sscanf(tahunStr, "%d", &startYear)
	}

	var kelasNama string
	_ = h.DB.QueryRow(`SELECT nama FROM kelas WHERE id = ?`, kelasID).Scan(&kelasNama)

	type srow struct {
		id   int64
		nis  string
		nama string
	}
	var list []srow
	srows, _ := h.DB.Query(`SELECT id, COALESCE(nis,''), nama FROM santri WHERE kelas_id = ? AND is_active = 1 ORDER BY nama`, kelasID)
	if srows != nil {
		for srows.Next() {
			var s srow
			_ = srows.Scan(&s.id, &s.nis, &s.nama)
			list = append(list, s)
		}
		srows.Close()
	}
	paid := map[int64]map[int]bool{}
	prows, _ := h.DB.Query(`
		SELECT sp.santri_id, sp.bulan FROM spp sp JOIN santri s ON s.id = sp.santri_id
		WHERE s.kelas_id = ? AND sp.lunas = 1 AND ((sp.tahun = ? AND sp.bulan >= 7) OR (sp.tahun = ? AND sp.bulan <= 6))`,
		kelasID, startYear, startYear+1)
	if prows != nil {
		for prows.Next() {
			var sid int64
			var bln int
			_ = prows.Scan(&sid, &bln)
			if paid[sid] == nil {
				paid[sid] = map[int]bool{}
			}
			paid[sid][bln] = true
		}
		prows.Close()
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := "SPP"
	si, _ := f.NewSheet(sheet)
	f.SetActiveSheet(si)
	f.DeleteSheet("Sheet1")

	f.SetCellValue(sheet, "A1", "REKAP PEMBAYARAN SPP")
	f.SetCellValue(sheet, "A2", fmt.Sprintf("Kelas: %s   |   Tahun Ajaran: %d/%d   |   ✓ = Lunas", kelasNama, startYear, startYear+1))

	headerRow := 4
	setCell := func(c, rownum int, v interface{}) {
		name, _ := excelize.CoordinatesToCellName(c, rownum)
		f.SetCellValue(sheet, name, v)
	}
	setCell(1, headerRow, "No")
	setCell(2, headerRow, "NIS")
	setCell(3, headerRow, "Nama")
	for i, b := range urutanBulanTA {
		setCell(4+i, headerRow, bulanSingkat[b-1])
	}
	totalCol := 16
	setCell(totalCol, headerRow, "Lunas")

	hStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	hs, _ := excelize.CoordinatesToCellName(1, headerRow)
	he, _ := excelize.CoordinatesToCellName(totalCol, headerRow)
	f.SetCellStyle(sheet, hs, he, hStyle)

	rownum := headerRow + 1
	for i, s := range list {
		setCell(1, rownum, i+1)
		setCell(2, rownum, s.nis)
		setCell(3, rownum, s.nama)
		cnt := 0
		for j, b := range urutanBulanTA {
			if paid[s.id][b] {
				setCell(4+j, rownum, "✓")
				cnt++
			}
		}
		setCell(totalCol, rownum, fmt.Sprintf("%d/12", cnt))
		rownum++
	}

	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 14)
	f.SetColWidth(sheet, "C", "C", 26)
	f.SetColWidth(sheet, "D", "O", 5)

	filename := fmt.Sprintf("SPP_%s_%d-%d.xlsx", sanitize(kelasNama), startYear, startYear+1)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := f.Write(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
	}
}
