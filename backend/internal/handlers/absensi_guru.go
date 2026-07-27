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

// GET /absensi-guru?tanggal=YYYY-MM-DD — semua guru + status pada tanggal tsb.
func (h *Handler) GetAbsensiGuru(w http.ResponseWriter, r *http.Request) {
	tanggal := r.URL.Query().Get("tanggal")
	if tanggal == "" {
		tanggal = time.Now().Format("2006-01-02")
	}
	rows, err := h.DB.Query(`
		SELECT g.id, g.nama, a.status, a.keterangan
		FROM guru g
		LEFT JOIN absensi_guru a ON a.guru_id = g.id AND a.tanggal = ?
		ORDER BY g.nama`, tanggal)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()
	type item struct {
		GuruID     int64   `json:"guru_id"`
		Nama       string  `json:"nama"`
		Status     string  `json:"status"`
		Keterangan *string `json:"keterangan"`
	}
	items := []item{}
	for rows.Next() {
		var it item
		var status *string
		_ = rows.Scan(&it.GuruID, &it.Nama, &status, &it.Keterangan)
		if status != nil {
			it.Status = *status
		}
		items = append(items, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"tanggal": tanggal, "items": items})
}

type absensiGuruBatch struct {
	Tanggal string `json:"tanggal"`
	Items   []struct {
		GuruID     int64   `json:"guru_id"`
		Status     string  `json:"status"`
		Keterangan *string `json:"keterangan"`
	} `json:"items"`
}

// POST /absensi-guru/batch — upsert absensi seluruh guru untuk satu tanggal.
func (h *Handler) SaveAbsensiGuru(w http.ResponseWriter, r *http.Request) {
	var batch absensiGuruBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return
	}
	if batch.Tanggal == "" || len(batch.Items) == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "tanggal dan items wajib")
		return
	}
	if _, err := time.Parse("2006-01-02", batch.Tanggal); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
		return
	}

	claims := middleware.ClaimsFrom(r)
	var userID interface{}
	if claims != nil {
		userID = claims.UserID
	}

	tx, err := h.DB.Begin()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO absensi_guru (guru_id, tanggal, status, keterangan, created_by)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), keterangan = VALUES(keterangan)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer stmt.Close()

	saved := 0
	for _, it := range batch.Items {
		if !validStatus[it.Status] {
			httpx.Error(w, http.StatusBadRequest, "INVALID_STATUS", "Status harus hadir/izin/sakit/alpha")
			return
		}
		if _, err := stmt.Exec(it.GuruID, batch.Tanggal, it.Status, it.Keterangan, userID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		saved++
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"saved": saved, "tanggal": batch.Tanggal})
}

// rentangFromTo membaca from/to (YYYY-MM-DD); default = bulan berjalan.
func rentangFromTo(r *http.Request) (string, string, error) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		now := time.Now()
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		from = first.Format("2006-01-02")
		to = first.AddDate(0, 1, -1).Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return "", "", err
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return "", "", err
	}
	return from, to, nil
}

type rekapGuruRow struct {
	GuruID int64  `json:"guru_id"`
	Nama   string `json:"nama"`
	Hadir  int    `json:"hadir"`
	Izin   int    `json:"izin"`
	Sakit  int    `json:"sakit"`
	Alpha  int    `json:"alpha"`
	Total  int    `json:"total"`
}

// rekapGuruData: jumlah H/I/S/A per guru pada rentang [from,to].
func (h *Handler) rekapGuruData(from, to string) ([]rekapGuruRow, error) {
	rows, err := h.DB.Query(`
		SELECT g.id, g.nama,
		       COALESCE(SUM(a.status = 'hadir'), 0),
		       COALESCE(SUM(a.status = 'izin'),  0),
		       COALESCE(SUM(a.status = 'sakit'), 0),
		       COALESCE(SUM(a.status = 'alpha'), 0)
		FROM guru g
		LEFT JOIN absensi_guru a ON a.guru_id = g.id AND a.tanggal BETWEEN ? AND ?
		GROUP BY g.id, g.nama
		ORDER BY g.nama`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rekapGuruRow{}
	for rows.Next() {
		var x rekapGuruRow
		_ = rows.Scan(&x.GuruID, &x.Nama, &x.Hadir, &x.Izin, &x.Sakit, &x.Alpha)
		x.Total = x.Hadir + x.Izin + x.Sakit + x.Alpha
		out = append(out, x)
	}
	return out, nil
}

// GET /absensi-guru/rekap?from=&to=
func (h *Handler) RekapAbsensiGuru(w http.ResponseWriter, r *http.Request) {
	from, to, err := rentangFromTo(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
		return
	}
	data, err := h.rekapGuruData(from, to)
	if err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"from": from, "to": to, "rows": data})
}

// GET /absensi-guru/export?from=&to=&label= — rekap ke Excel.
func (h *Handler) ExportAbsensiGuru(w http.ResponseWriter, r *http.Request) {
	from, to, err := rentangFromTo(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
		return
	}
	label := r.URL.Query().Get("label")
	if label == "" {
		label = from + " s/d " + to
	}
	data, err := h.rekapGuruData(from, to)
	if err != nil {
		dbErr(w, err)
		return
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Rekap Absensi Guru"
	idx, _ := f.NewSheet(sheet)
	f.SetActiveSheet(idx)
	f.DeleteSheet("Sheet1")

	f.SetCellValue(sheet, "A1", "REKAP ABSENSI GURU")
	f.SetCellValue(sheet, "A2", "Periode: "+label)
	f.SetCellValue(sheet, "A3", "Keterangan: H=Hadir, I=Izin, S=Sakit, A=Alpha")

	headerRow := 5
	setCell := func(c, rownum int, v interface{}) {
		name, _ := excelize.CoordinatesToCellName(c, rownum)
		f.SetCellValue(sheet, name, v)
	}
	heads := []string{"No", "Nama", "Hadir", "Izin", "Sakit", "Alpha", "Total"}
	for i, hh := range heads {
		setCell(i+1, headerRow, hh)
	}
	hStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	hs, _ := excelize.CoordinatesToCellName(1, headerRow)
	he, _ := excelize.CoordinatesToCellName(len(heads), headerRow)
	f.SetCellStyle(sheet, hs, he, hStyle)

	rownum := headerRow + 1
	for i, x := range data {
		setCell(1, rownum, i+1)
		setCell(2, rownum, x.Nama)
		setCell(3, rownum, x.Hadir)
		setCell(4, rownum, x.Izin)
		setCell(5, rownum, x.Sakit)
		setCell(6, rownum, x.Alpha)
		setCell(7, rownum, x.Total)
		rownum++
	}

	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 28)

	filename := fmt.Sprintf("RekapAbsensiGuru_%s_%s.xlsx", from, to)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := f.Write(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
	}
}
