package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
	"sim-madrasah/backend/internal/models"
)

func validNilai(p *float64) bool {
	if p == nil {
		return true // boleh kosong
	}
	return *p >= 0 && *p <= 100
}

// GET /nilai?kelas_id=&mata_pelajaran_id=&periode_id=
func (h *Handler) GetNilai(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	mapelID := r.URL.Query().Get("mata_pelajaran_id")
	periodeID := r.URL.Query().Get("periode_id")
	if kelasID == "" || mapelID == "" || periodeID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id, mata_pelajaran_id, periode_id wajib")
		return
	}

	rows, err := h.DB.Query(`
		SELECT s.id, COALESCE(s.nis,''), s.nama, n.tugas, n.uts, n.uas, n.nilai_akhir
		FROM santri s
		LEFT JOIN nilai n ON n.santri_id = s.id AND n.mata_pelajaran_id = ? AND n.periode_id = ?
		WHERE s.kelas_id = ? AND s.is_active = 1
		ORDER BY s.nama`, mapelID, periodeID, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	items := []models.NilaiItem{}
	for rows.Next() {
		var it models.NilaiItem
		_ = rows.Scan(&it.SantriID, &it.NIS, &it.Nama, &it.Tugas, &it.UTS, &it.UAS, &it.NilaiAkhir)
		items = append(items, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"kelas_id": kelasID, "mata_pelajaran_id": mapelID, "periode_id": periodeID,
		"bobot": map[string]float64{"tugas": models.BobotTugas, "uts": models.BobotUTS, "uas": models.BobotUAS},
		"items": items,
	})
}

// POST /nilai/batch — upsert + hitung nilai akhir (Tugas 30% + UTS 30% + UAS 40%).
func (h *Handler) SaveNilai(w http.ResponseWriter, r *http.Request) {
	var batch models.NilaiBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return
	}
	if batch.KelasID == 0 || batch.MataPelajaranID == 0 || batch.PeriodeID == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id, mata_pelajaran_id, periode_id wajib")
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
		INSERT INTO nilai (santri_id, mata_pelajaran_id, periode_id, tugas, uts, uas, nilai_akhir, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE tugas=VALUES(tugas), uts=VALUES(uts), uas=VALUES(uas), nilai_akhir=VALUES(nilai_akhir)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer stmt.Close()

	saved := 0
	for _, it := range batch.Items {
		if !validNilai(it.Tugas) || !validNilai(it.UTS) || !validNilai(it.UAS) {
			httpx.Error(w, http.StatusBadRequest, "INVALID_NILAI", "Nilai harus antara 0 dan 100")
			return
		}
		akhir := models.HitungNilaiAkhir(it.Tugas, it.UTS, it.UAS)
		if _, err := stmt.Exec(it.SantriID, batch.MataPelajaranID, batch.PeriodeID,
			it.Tugas, it.UTS, it.UAS, akhir, userID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		saved++
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"saved": saved})
}

// recalcNilaiAkhir menulis ulang nilai_akhir baris nilai dari komponen tersimpan
// (Tugas 30% + UTS 30% + UAS 40%; komponen NULL dianggap 0). Baris harus sudah ada.
func recalcNilaiAkhir(tx *sql.Tx, santriID, mapelID, periodeID int64) error {
	_, err := tx.Exec(`
        UPDATE nilai
           SET nilai_akhir = ROUND(COALESCE(tugas,0)*0.30 + COALESCE(uts,0)*0.30 + COALESCE(uas,0)*0.40, 2)
         WHERE santri_id = ? AND mata_pelajaran_id = ? AND periode_id = ?`,
		santriID, mapelID, periodeID)
	return err
}
