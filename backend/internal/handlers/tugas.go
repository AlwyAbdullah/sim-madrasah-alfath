package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

type tugasReq struct {
	KelasID         int64   `json:"kelas_id"`
	MataPelajaranID *int64  `json:"mata_pelajaran_id"` // nullable di DB; bot mewajibkan
	Deskripsi       string  `json:"deskripsi"`
	TanggalDiberikan string  `json:"tanggal_diberikan"` // opsional; kosong = hari ini
	Tenggat         *string `json:"tenggat"`           // opsional; null = tanpa tenggat
}

// POST /tugas — umumkan satu tugas/PR untuk satu kelas (created_by = guru dari JWT).
func (h *Handler) CreateTugas(w http.ResponseWriter, r *http.Request) {
	var req tugasReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return
	}
	deskripsi := strings.TrimSpace(req.Deskripsi)
	if req.KelasID == 0 || deskripsi == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id dan deskripsi wajib")
		return
	}
	if len([]rune(deskripsi)) > 500 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Deskripsi maksimal 500 karakter")
		return
	}
	tglDiberikan := strings.TrimSpace(req.TanggalDiberikan)
	if tglDiberikan == "" {
		tglDiberikan = time.Now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", tglDiberikan); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal_diberikan harus YYYY-MM-DD")
		return
	}

	// tenggat: opsional. Nil atau string kosong → NULL. Bila diisi → validasi.
	var tenggat interface{}
	if req.Tenggat != nil && strings.TrimSpace(*req.Tenggat) != "" {
		t := strings.TrimSpace(*req.Tenggat)
		if _, err := time.Parse("2006-01-02", t); err != nil {
			httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tenggat harus YYYY-MM-DD")
			return
		}
		tenggat = t
	}

	claims := middleware.ClaimsFrom(r)
	var userID interface{}
	if claims != nil {
		userID = claims.UserID
	}

	res, err := h.DB.Exec(
		`INSERT INTO tugas (kelas_id, mata_pelajaran_id, deskripsi, tanggal_diberikan, tenggat, created_by)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		req.KelasID, req.MataPelajaranID, deskripsi, tglDiberikan, tenggat, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"id": id, "kelas_id": req.KelasID, "mata_pelajaran_id": req.MataPelajaranID,
		"deskripsi": deskripsi, "tanggal_diberikan": tglDiberikan, "tenggat": tenggat,
	})
}

// GET /tugas?kelas_id=&aktif= — daftar tugas satu kelas (aktif=1 → belum lewat tenggat).
func (h *Handler) GetTugasList(w http.ResponseWriter, r *http.Request) {
	kelasID := r.URL.Query().Get("kelas_id")
	if kelasID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "kelas_id wajib")
		return
	}
	where := "t.kelas_id = ?"
	if r.URL.Query().Get("aktif") == "1" {
		where += " AND (t.tenggat IS NULL OR t.tenggat >= CURDATE())"
	}

	rows, err := h.DB.Query(`
		SELECT t.id, t.kelas_id, t.mata_pelajaran_id, COALESCE(mp.nama,''),
		       t.deskripsi, DATE_FORMAT(t.tanggal_diberikan,'%Y-%m-%d'),
		       DATE_FORMAT(t.tenggat,'%Y-%m-%d'),
		       DATE_FORMAT(t.created_at,'%Y-%m-%d %H:%i:%s')
		FROM tugas t
		LEFT JOIN mata_pelajaran mp ON mp.id = t.mata_pelajaran_id
		WHERE `+where+`
		ORDER BY t.tanggal_diberikan DESC, t.id DESC`, kelasID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	type tugasRow struct {
		ID               int64   `json:"id"`
		KelasID          int64   `json:"kelas_id"`
		MataPelajaranID  *int64  `json:"mata_pelajaran_id"`
		MapelNama        string  `json:"mapel_nama"`
		Deskripsi        string  `json:"deskripsi"`
		TanggalDiberikan string  `json:"tanggal_diberikan"`
		Tenggat          *string `json:"tenggat"`
		CreatedAt        string  `json:"created_at"`
	}
	out := []tugasRow{}
	for rows.Next() {
		var t tugasRow
		if err := rows.Scan(&t.ID, &t.KelasID, &t.MataPelajaranID, &t.MapelNama,
			&t.Deskripsi, &t.TanggalDiberikan, &t.Tenggat, &t.CreatedAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
