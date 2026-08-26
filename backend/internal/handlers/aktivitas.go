package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"sim-madrasah/backend/internal/httpx"
)

type entriAktivitas struct {
	ID        int64   `json:"id"`
	UserID    *int64  `json:"user_id"`
	Username  string  `json:"username"`
	Nama      string  `json:"nama"`
	Aksi      string  `json:"aksi"`
	Entitas   *string `json:"entitas"`
	EntitasID *string `json:"entitas_id"`
	Ringkasan string  `json:"ringkasan"`
	Rincian   *string `json:"rincian"`
	IP        *string `json:"ip"`
	CreatedAt string  `json:"created_at"`
}

// GET /aktivitas?user_id=&aksi=&dari=&sampai=&limit=&offset=
//
// Lini masa perubahan sistem. Disaring, bukan diambil semuanya sekaligus —
// setelah setahun isinya puluhan ribu baris.
func (h *Handler) ListAktivitas(w http.ResponseWriter, r *http.Request) {
	q := `SELECT id, user_id, username, nama, aksi, entitas, entitas_id, ringkasan, rincian, ip, created_at
	      FROM log_aktivitas WHERE 1 = 1`
	args := []interface{}{}

	if v := r.URL.Query().Get("user_id"); v != "" {
		q += ` AND user_id = ?`
		args = append(args, v)
	}
	if v := r.URL.Query().Get("aksi"); v != "" {
		q += ` AND aksi = ?`
		args = append(args, v)
	}
	if v := r.URL.Query().Get("dari"); v != "" {
		q += ` AND created_at >= ?`
		args = append(args, v+" 00:00:00")
	}
	if v := r.URL.Query().Get("sampai"); v != "" {
		q += ` AND created_at <= ?`
		args = append(args, v+" 23:59:59")
	}
	if v := strings.TrimSpace(r.URL.Query().Get("cari")); v != "" {
		q += ` AND (ringkasan LIKE ? OR nama LIKE ? OR username LIKE ?)`
		pola := "%" + v + "%"
		args = append(args, pola, pola, pola)
	}

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	q += ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.DB.Query(q, args...)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	out := []entriAktivitas{}
	for rows.Next() {
		var e entriAktivitas
		var uid sql.NullInt64
		var ent, entID, rinc, ip sql.NullString
		if err := rows.Scan(&e.ID, &uid, &e.Username, &e.Nama, &e.Aksi,
			&ent, &entID, &e.Ringkasan, &rinc, &ip, &e.CreatedAt); err != nil {
			continue
		}
		if uid.Valid {
			v := uid.Int64
			e.UserID = &v
		}
		if ent.Valid {
			e.Entitas = &ent.String
		}
		if entID.Valid {
			e.EntitasID = &entID.String
		}
		if rinc.Valid {
			e.Rincian = &rinc.String
		}
		if ip.Valid {
			e.IP = &ip.String
		}
		out = append(out, e)
	}

	// pilihan penyaring diambil dari isi tabel, bukan dipaku di frontend, supaya
	// aksi baru otomatis muncul tanpa mengubah dua tempat
	aksi := []string{}
	if arows, err := h.DB.Query(`SELECT DISTINCT aksi FROM log_aktivitas ORDER BY aksi`); err == nil {
		for arows.Next() {
			var a string
			if arows.Scan(&a) == nil {
				aksi = append(aksi, a)
			}
		}
		arows.Close()
	}

	pelaku := []map[string]interface{}{}
	if prows, err := h.DB.Query(`
		SELECT user_id, MAX(nama) FROM log_aktivitas
		WHERE user_id IS NOT NULL GROUP BY user_id ORDER BY MAX(nama)`); err == nil {
		for prows.Next() {
			var id int64
			var nama string
			if prows.Scan(&id, &nama) == nil {
				pelaku = append(pelaku, map[string]interface{}{"user_id": id, "nama": strings.TrimSpace(nama)})
			}
		}
		prows.Close()
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"items":  out,
		"aksi":   aksi,
		"pelaku": pelaku,
		"limit":  limit,
		"offset": offset,
	})
}
