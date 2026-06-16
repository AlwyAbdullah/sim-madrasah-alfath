package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/auth"
	"sim-madrasah/backend/internal/httpx"
)

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, username, nama, role, is_active FROM users ORDER BY username`)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()
	type u struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Nama     string `json:"nama"`
		Role     string `json:"role"`
		IsActive bool   `json:"is_active"`
	}
	out := []u{}
	for rows.Next() {
		var x u
		_ = rows.Scan(&x.ID, &x.Username, &x.Nama, &x.Role, &x.IsActive)
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type userReq struct {
	Username string `json:"username"`
	Nama     string `json:"nama"`
	Role     string `json:"role"`
	Password string `json:"password"`
	IsActive *bool  `json:"is_active"`
}

func validRole(r string) bool {
	return r == "admin" || r == "guru" || r == "kepala"
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req userReq
	if !decode(w, r, &req) {
		return
	}
	if req.Username == "" || req.Nama == "" || req.Password == "" || !validRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Username, nama, password, dan role (admin/guru/kepala) wajib")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO users (username, password_hash, nama, role) VALUES (?, ?, ?, ?)`,
		req.Username, hash, req.Nama, req.Role)
	if err != nil {
		dbErr(w, err)
		return
	}
	id, _ := res.LastInsertId()
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// UpdateUser: ubah nama/role/status. Password hanya diubah bila diisi (reset).
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req userReq
	if !decode(w, r, &req) {
		return
	}
	if req.Nama == "" || !validRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Nama dan role wajib")
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	if _, err := h.DB.Exec(`UPDATE users SET nama = ?, role = ?, is_active = ? WHERE id = ?`,
		req.Nama, req.Role, active, id); err != nil {
		dbErr(w, err)
		return
	}
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Gagal memproses password")
			return
		}
		if _, err := h.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id); err != nil {
			dbErr(w, err)
			return
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// DeleteUser: nonaktifkan (soft-delete) agar referensi audit tetap utuh.
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`UPDATE users SET is_active = 0 WHERE id = ?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}
