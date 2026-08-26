package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/audit"
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

	// Siapa yang terakhir menyentuh absensi kelas ini pada tanggal tsb — supaya
	// guru langsung tahu datanya sudah diisi orang lain sebelum menimpanya.
	resp := map[string]interface{}{"kelas_id": kelasID, "tanggal": tanggal, "items": items}
	var oleh sql.NullString
	var kapan sql.NullString
	if err := h.DB.QueryRow(`
		SELECT u.nama, MAX(a.updated_at)
		FROM absensi a
		JOIN santri s ON s.id = a.santri_id
		LEFT JOIN users u ON u.id = a.updated_by
		WHERE s.kelas_id = ? AND a.tanggal = ? AND a.updated_by IS NOT NULL
		GROUP BY u.nama
		ORDER BY MAX(a.updated_at) DESC LIMIT 1`, kelasID, tanggal).Scan(&oleh, &kapan); err == nil {
		if oleh.Valid {
			resp["terakhir_diubah_oleh"] = strings.TrimSpace(oleh.String)
		}
		if kapan.Valid {
			resp["terakhir_diubah_pada"] = kapan.String
		}
	}

	httpx.JSON(w, http.StatusOK, resp)
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

	// updated_by diisi juga di cabang ON DUPLICATE KEY UPDATE — tanpa itu yang
	// tercatat selamanya cuma pembuat pertama, bukan pengubah terakhir.
	stmt, err := tx.Prepare(`
		INSERT INTO absensi (santri_id, tanggal, status, keterangan, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), keterangan = VALUES(keterangan),
		                        updated_by = VALUES(updated_by)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer stmt.Close()

	saved := 0
	ids := make([]int64, 0, len(batch.Items))
	for _, it := range batch.Items {
		if !validStatus[it.Status] {
			httpx.Error(w, http.StatusBadRequest, "INVALID_STATUS", "Status harus hadir/izin/sakit/alpha")
			return
		}
		if _, err := stmt.Exec(it.SantriID, batch.Tanggal, it.Status, it.Keterangan, userID, userID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		ids = append(ids, it.SantriID)
		saved++
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	audit.Catat(h.DB, r, audit.Entri{
		Aksi:      audit.SimpanAbsensi,
		Entitas:   "absensi",
		EntitasID: batch.Tanggal,
		Ringkasan: fmt.Sprintf("Menyimpan absensi %s — %d santri, %s",
			h.namaKelasDariSantri(ids), saved, batch.Tanggal),
	})

	httpx.JSON(w, http.StatusOK, map[string]interface{}{"saved": saved, "tanggal": batch.Tanggal})
}

// namaKelasDariSantri menyusun label kelas untuk ringkasan log. Satu batch
// biasanya satu kelas; kalau ternyata lebih, semuanya disebut agar ringkasannya
// tidak menyesatkan.
func (h *Handler) namaKelasDariSantri(ids []int64) string {
	if len(ids) == 0 {
		return "-"
	}
	tanda := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		tanda[i] = "?"
		args[i] = id
	}
	rows, err := h.DB.Query(`
		SELECT DISTINCT k.nama FROM santri s JOIN kelas k ON k.id = s.kelas_id
		WHERE s.id IN (`+strings.Join(tanda, ",")+`) ORDER BY k.nama`, args...)
	if err != nil {
		return "-"
	}
	defer rows.Close()
	var nama []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			nama = append(nama, n)
		}
	}
	if len(nama) == 0 {
		return "-"
	}
	return strings.Join(nama, ", ")
}
