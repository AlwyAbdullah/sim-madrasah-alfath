package handlers

import (
	"encoding/json"
	"net/http"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

// GET /nilai/tugas?kelas_id=&mata_pelajaran_id=&periode_id=
// Per santri: daftar {ke, nilai} + rata. Juga next_ke untuk auto-increment bot.
func (h *Handler) GetTugas(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	mapelID := r.URL.Query().Get("mata_pelajaran_id")
	periodeID := r.URL.Query().Get("periode_id")
	if kelasID == "" || mapelID == "" || periodeID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id, mata_pelajaran_id, periode_id wajib")
		return
	}

	rows, err := h.DB.Query(`
        SELECT s.id, s.nama, nt.ke, nt.nilai
        FROM santri s
        LEFT JOIN nilai_tugas nt
          ON nt.santri_id = s.id AND nt.mata_pelajaran_id = ? AND nt.periode_id = ?
        WHERE s.kelas_id = ? AND s.is_active = 1
        ORDER BY s.nama, nt.ke`, mapelID, periodeID, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	type tugasItem struct {
		Ke    int     `json:"ke"`
		Nilai float64 `json:"nilai"`
	}
	type santriTugas struct {
		SantriID int64       `json:"santri_id"`
		Nama     string      `json:"nama"`
		List     []tugasItem `json:"list"`
		Rata     *float64    `json:"rata"`
	}

	order := []int64{}
	bySantri := map[int64]*santriTugas{}
	maxKe := 0
	for rows.Next() {
		var sid int64
		var nama string
		var ke *int
		var nilai *float64
		_ = rows.Scan(&sid, &nama, &ke, &nilai)
		st, ok := bySantri[sid]
		if !ok {
			st = &santriTugas{SantriID: sid, Nama: nama, List: []tugasItem{}}
			bySantri[sid] = st
			order = append(order, sid)
		}
		if ke != nil && nilai != nil {
			st.List = append(st.List, tugasItem{Ke: *ke, Nilai: *nilai})
			if *ke > maxKe {
				maxKe = *ke
			}
		}
	}
	out := make([]*santriTugas, 0, len(order))
	for _, sid := range order {
		st := bySantri[sid]
		if n := len(st.List); n > 0 {
			sum := 0.0
			for _, t := range st.List {
				sum += t.Nilai
			}
			avg := float64(int(sum/float64(n)*100+0.5)) / 100
			st.Rata = &avg
		}
		out = append(out, st)
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"kelas_id": kelasID, "mata_pelajaran_id": mapelID, "periode_id": periodeID,
		"next_ke": maxKe + 1,
		"items":   out,
	})
}

type tugasBatchReq struct {
	KelasID         int64 `json:"kelas_id"`
	MataPelajaranID int64 `json:"mata_pelajaran_id"`
	PeriodeID       int64 `json:"periode_id"`
	Ke              *int  `json:"ke"` // opsional; kosong = next_ke
	Items           []struct {
		SantriID int64    `json:"santri_id"`
		Nilai    *float64 `json:"nilai"`
	} `json:"items"`
}

// POST /nilai/tugas/batch — simpan satu Tugas ke-N untuk satu kelas+mapel+periode,
// lalu re-average kolom nilai.tugas dan hitung ulang nilai_akhir.
func (h *Handler) SaveTugasBatch(w http.ResponseWriter, r *http.Request) {
	var req tugasBatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return
	}
	if req.KelasID == 0 || req.MataPelajaranID == 0 || req.PeriodeID == 0 || len(req.Items) == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id, mata_pelajaran_id, periode_id, items wajib")
		return
	}
	for _, it := range req.Items {
		if it.Nilai == nil || *it.Nilai < 0 || *it.Nilai > 100 {
			httpx.Error(w, http.StatusBadRequest, "INVALID_NILAI", "Nilai tugas harus 0-100")
			return
		}
	}

	claims := middleware.ClaimsFrom(r)
	var userID interface{}
	if claims != nil {
		userID = claims.UserID
	}

	// Begin transaction first to avoid TOCTOU race on ke auto-increment
	tx, err := h.DB.Begin()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback()

	// Tentukan ke: pakai req.Ke bila ada; jika tidak, max(ke)+1 untuk kelas+mapel+periode.
	ke := 0
	if req.Ke != nil {
		ke = *req.Ke
	} else {
		_ = tx.QueryRow(`
            SELECT COALESCE(MAX(nt.ke),0)+1
            FROM nilai_tugas nt JOIN santri s ON s.id = nt.santri_id
            WHERE s.kelas_id = ? AND nt.mata_pelajaran_id = ? AND nt.periode_id = ?`,
			req.KelasID, req.MataPelajaranID, req.PeriodeID).Scan(&ke)
		if ke == 0 {
			ke = 1
		}
	}

	insTugas, err := tx.Prepare(`
        INSERT INTO nilai_tugas (santri_id, mata_pelajaran_id, periode_id, ke, nilai, created_by)
        VALUES (?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE nilai = VALUES(nilai)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer insTugas.Close()

	// Upsert baris nilai dengan tugas = rata-rata (dihitung di Go lalu di-pass sbg
	// parameter — MySQL tidak mengizinkan subquery di dalam VALUES(...)).
	upNilai, err := tx.Prepare(`
        INSERT INTO nilai (santri_id, mata_pelajaran_id, periode_id, tugas, nilai_akhir, created_by)
        VALUES (?, ?, ?, ?, 0, ?)
        ON DUPLICATE KEY UPDATE tugas = VALUES(tugas)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer upNilai.Close()

	saved := 0
	for _, it := range req.Items {
		// 1. upsert baris detail tugas
		if _, err := insTugas.Exec(it.SantriID, req.MataPelajaranID, req.PeriodeID, ke, it.Nilai, userID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		// 2. hitung rata-rata terbaru untuk santri tsb
		var avg float64
		if err := tx.QueryRow(
			`SELECT ROUND(AVG(nilai),2) FROM nilai_tugas WHERE santri_id=? AND mata_pelajaran_id=? AND periode_id=?`,
			it.SantriID, req.MataPelajaranID, req.PeriodeID).Scan(&avg); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		// 3. upsert baris nilai dengan tugas = rata-rata
		if _, err := upNilai.Exec(it.SantriID, req.MataPelajaranID, req.PeriodeID, avg, userID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		// 4. hitung ulang nilai_akhir dari komponen tersimpan
		if err := recalcNilaiAkhir(tx, it.SantriID, req.MataPelajaranID, req.PeriodeID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		saved++
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"saved": saved, "ke": ke})
}
