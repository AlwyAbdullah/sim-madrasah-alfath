package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
	"sim-madrasah/backend/internal/models"
)

var validStatus = map[string]bool{"hadir": true, "izin": true, "sakit": true, "alpha": true}

// GET /absensi?kelas_id=&tanggal=
// Mengembalikan daftar santri kelas + status absensi pada tanggal tsb (jika ada).
func (h *Handler) GetAbsensi(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	tanggal := r.URL.Query().Get("tanggal")
	if kelasID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id wajib")
		return
	}
	if tanggal == "" {
		tanggal = time.Now().Format("2006-01-02")
	}

	rows, err := h.DB.Query(`
		SELECT s.id, COALESCE(s.nis,''), s.nama,
		       a.status, a.keterangan
		FROM santri s
		LEFT JOIN absensi a ON a.santri_id = s.id AND a.tanggal = ?
		WHERE s.kelas_id = ? AND s.is_active = 1
		ORDER BY s.nama`, tanggal, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	items := []models.AbsensiItem{}
	for rows.Next() {
		var it models.AbsensiItem
		var status *string
		_ = rows.Scan(&it.SantriID, &it.NIS, &it.Nama, &status, &it.Keterangan)
		if status != nil {
			it.Status = *status
		}
		items = append(items, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"kelas_id": kelasID, "tanggal": tanggal, "items": items,
	})
}

// POST /absensi/batch — upsert seluruh kelas dalam satu transaksi.
func (h *Handler) SaveAbsensi(w http.ResponseWriter, r *http.Request) {
	var batch models.AbsensiBatch
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
		INSERT INTO absensi (santri_id, tanggal, status, keterangan, created_by)
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
		if _, err := stmt.Exec(it.SantriID, batch.Tanggal, it.Status, it.Keterangan, userID); err != nil {
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
