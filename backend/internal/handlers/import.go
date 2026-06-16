package handlers

import (
	"net/http"
	"strings"

	"github.com/xuri/excelize/v2"

	"sim-madrasah/backend/internal/httpx"
)

type importError struct {
	Baris int    `json:"baris"`
	Pesan string `json:"pesan"`
}

// POST /santri/import  (multipart form, field "file" = .xlsx)
// Kolom diharapkan (header baris 1, urutan bebas): NIS, Nama, Kelas, JK/L-P
func (h *Handler) ImportSantri(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // maks 10MB
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Gagal membaca form")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "NO_FILE", "File tidak ditemukan (field 'file')")
		return
	}
	defer file.Close()

	xl, err := excelize.OpenReader(file)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_FILE", "File Excel tidak valid")
		return
	}
	defer xl.Close()

	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		httpx.Error(w, http.StatusBadRequest, "EMPTY", "Sheet kosong atau tidak ada data")
		return
	}

	// petakan kolom dari header
	header := rows[0]
	col := map[string]int{}
	for i, c := range header {
		k := strings.ToLower(strings.TrimSpace(c))
		switch {
		case strings.Contains(k, "nis"):
			col["nis"] = i
		case strings.Contains(k, "nama"):
			col["nama"] = i
		case strings.Contains(k, "kelas"):
			col["kelas"] = i
		case strings.Contains(k, "jk") || strings.Contains(k, "kelamin") || k == "l/p":
			col["jk"] = i
		}
	}
	if _, ok := col["nama"]; !ok {
		httpx.Error(w, http.StatusBadRequest, "NO_NAMA", "Kolom 'Nama' tidak ditemukan di header")
		return
	}
	if _, ok := col["kelas"]; !ok {
		httpx.Error(w, http.StatusBadRequest, "NO_KELAS", "Kolom 'Kelas' tidak ditemukan di header")
		return
	}

	// map nama kelas -> id
	kelasMap := map[string]int64{}
	krows, _ := h.DB.Query(`SELECT id, nama FROM kelas`)
	if krows != nil {
		for krows.Next() {
			var id int64
			var nama string
			_ = krows.Scan(&id, &nama)
			kelasMap[strings.ToLower(strings.TrimSpace(nama))] = id
		}
		krows.Close()
	}

	get := func(row []string, key string) string {
		if i, ok := col[key]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	stmt, err := h.DB.Prepare(`
		INSERT INTO santri (nis, nama, jenis_kelamin, kelas_id, is_active)
		VALUES (?, ?, ?, ?, 1)
		ON DUPLICATE KEY UPDATE nama=VALUES(nama), jenis_kelamin=VALUES(jenis_kelamin), kelas_id=VALUES(kelas_id), is_active=1`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer stmt.Close()

	errs := []importError{}
	saved := 0
	for i := 1; i < len(rows); i++ {
		baris := i + 1 // nomor baris di Excel (1-based, +header)
		nama := get(rows[i], "nama")
		if nama == "" {
			continue // lewati baris kosong
		}
		kelasNama := get(rows[i], "kelas")
		kid, ok := kelasMap[strings.ToLower(kelasNama)]
		if !ok {
			errs = append(errs, importError{Baris: baris, Pesan: "Kelas '" + kelasNama + "' tidak ditemukan"})
			continue
		}
		jk := strings.ToUpper(get(rows[i], "jk"))
		if jk != "L" && jk != "P" {
			jk = "L" // default
		}
		nis := get(rows[i], "nis")
		var nisVal interface{}
		if nis != "" {
			nisVal = nis
		}
		if _, err := stmt.Exec(nisVal, nama, jk, kid); err != nil {
			errs = append(errs, importError{Baris: baris, Pesan: err.Error()})
			continue
		}
		saved++
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"tersimpan": saved,
		"gagal":     len(errs),
		"errors":    errs,
	})
}
