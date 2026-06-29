package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

type catatanReq struct {
	SantriID int64  `json:"santri_id"`
	Teks     string `json:"teks"`
	Tanggal  string `json:"tanggal"` // opsional; kosong = hari ini
}

// POST /catatan — simpan satu catatan untuk satu santri (created_by = guru dari JWT).
func (h *Handler) CreateCatatan(w http.ResponseWriter, r *http.Request) {
	var req catatanReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid")
		return
	}
	teks := strings.TrimSpace(req.Teks)
	if req.SantriID == 0 || teks == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "santri_id dan teks wajib")
		return
	}
	if len([]rune(teks)) > 500 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Teks maksimal 500 karakter")
		return
	}
	tanggal := strings.TrimSpace(req.Tanggal)
	if tanggal == "" {
		tanggal = time.Now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", tanggal); err != nil {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
		return
	}

	claims := middleware.ClaimsFrom(r)
	var userID interface{}
	if claims != nil {
		userID = claims.UserID
	}

	res, err := h.DB.Exec(
		`INSERT INTO catatan (santri_id, tanggal, teks, created_by) VALUES (?, ?, ?, ?)`,
		req.SantriID, tanggal, teks, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"id": id, "santri_id": req.SantriID, "tanggal": tanggal, "teks": teks,
	})
}

// GET /catatan?santri_id=&limit= — catatan terbaru satu santri (limit default 5).
func (h *Handler) GetCatatan(w http.ResponseWriter, r *http.Request) {
	santriID := r.URL.Query().Get("santri_id")
	if santriID == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "santri_id wajib")
		return
	}
	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	rows, err := h.DB.Query(`
		SELECT id, santri_id, DATE_FORMAT(tanggal,'%Y-%m-%d'), teks,
		       created_by, DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s')
		FROM catatan
		WHERE santri_id = ?
		ORDER BY tanggal DESC, id DESC
		LIMIT ?`, santriID, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	type catatanRow struct {
		ID        int64  `json:"id"`
		SantriID  int64  `json:"santri_id"`
		Tanggal   string `json:"tanggal"`
		Teks      string `json:"teks"`
		CreatedBy *int64 `json:"created_by"`
		CreatedAt string `json:"created_at"`
	}
	out := []catatanRow{}
	for rows.Next() {
		var c catatanRow
		if err := rows.Scan(&c.ID, &c.SantriID, &c.Tanggal, &c.Teks, &c.CreatedBy, &c.CreatedAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
