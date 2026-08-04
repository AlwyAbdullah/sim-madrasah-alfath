package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/middleware"
)

// newBatchID membuat UUID v4 tanpa dependensi tambahan.
func newBatchID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ts-%d-%d", time.Now().UnixNano(), time.Now().Unix())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // versi 4
	b[8] = (b[8] & 0x3f) | 0x80 // varian RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// selKey menandai satu sel SPP (santri + tahun kalender + bulan).
type selKey struct {
	SantriID int64
	Tahun    int
	Bulan    int
}

type nilaiSel struct {
	Lunas bool
	Ket   sql.NullString
	Ada   bool // baris spp sudah ada sebelumnya?
}

// ambilNilaiLama membaca kondisi sel SEBELUM diubah, untuk dicatat di riwayat.
// Satu query untuk semua santri pada rentang tahun ajaran, bukan per sel.
func ambilNilaiLama(tx *sql.Tx, santriIDs []int64, taStart int) (map[selKey]nilaiSel, error) {
	out := map[selKey]nilaiSel{}
	if len(santriIDs) == 0 {
		return out, nil
	}
	q := `SELECT santri_id, tahun, bulan, lunas, keterangan FROM spp
	      WHERE ((tahun = ? AND bulan >= 7) OR (tahun = ? AND bulan <= 6)) AND santri_id IN (?` +
		repeatParam(len(santriIDs)-1) + `)`
	args := []interface{}{taStart, taStart + 1}
	for _, id := range santriIDs {
		args = append(args, id)
	}
	rows, err := tx.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k selKey
		var v nilaiSel
		if err := rows.Scan(&k.SantriID, &k.Tahun, &k.Bulan, &v.Lunas, &v.Ket); err != nil {
			continue
		}
		v.Ada = true
		out[k] = v
	}
	return out, rows.Err()
}

func repeatParam(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += ",?"
	}
	return s
}

// GET /spp/riwayat?limit= — daftar aksi simpan/kembalikan terbaru.
func (h *Handler) ListRiwayatSPP(w http.ResponseWriter, r *http.Request) {
	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit < 1 || limit > 200 {
			limit = 30
		}
	}
	rows, err := h.DB.Query(`
		SELECT b.batch_id, b.aksi, b.jumlah_sel, b.tahun_ajaran, b.dikembalikan,
		       COALESCE(u.nama, '-'), b.created_at
		FROM spp_riwayat_batch b
		LEFT JOIN users u ON u.id = b.created_by
		ORDER BY b.created_at DESC, b.batch_id DESC
		LIMIT ?`, limit)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	type item struct {
		BatchID      string `json:"batch_id"`
		Aksi         string `json:"aksi"`
		JumlahSel    int    `json:"jumlah_sel"`
		TahunAjaran  int    `json:"tahun_ajaran"`
		Dikembalikan bool   `json:"dikembalikan"`
		Oleh         string `json:"oleh"`
		Waktu        string `json:"waktu"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var waktu time.Time
		if err := rows.Scan(&it.BatchID, &it.Aksi, &it.JumlahSel, &it.TahunAjaran,
			&it.Dikembalikan, &it.Oleh, &waktu); err != nil {
			continue
		}
		it.Waktu = waktu.Format("2006-01-02 15:04:05")
		out = append(out, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

// GET /spp/riwayat/{batch}/detail — rincian sel yang berubah pada satu aksi.
func (h *Handler) DetailRiwayatSPP(w http.ResponseWriter, r *http.Request) {
	batch := chi.URLParam(r, "batch")
	rows, err := h.DB.Query(`
		SELECT s.nama, rw.tahun, rw.bulan, rw.lunas_lama, rw.lunas_baru,
		       COALESCE(rw.ket_lama,''), COALESCE(rw.ket_baru,'')
		FROM spp_riwayat rw JOIN santri s ON s.id = rw.santri_id
		WHERE rw.batch_id = ?
		ORDER BY s.nama, rw.tahun, rw.bulan`, batch)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	type item struct {
		Nama      string `json:"nama"`
		Tahun     int    `json:"tahun"`
		Bulan     int    `json:"bulan"`
		LunasLama *bool  `json:"lunas_lama"`
		LunasBaru bool   `json:"lunas_baru"`
		KetLama   string `json:"ket_lama"`
		KetBaru   string `json:"ket_baru"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var lama sql.NullBool
		if err := rows.Scan(&it.Nama, &it.Tahun, &it.Bulan, &lama, &it.LunasBaru,
			&it.KetLama, &it.KetBaru); err != nil {
			continue
		}
		if lama.Valid {
			v := lama.Bool
			it.LunasLama = &v
		}
		out = append(out, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

// POST /spp/riwayat/{batch}/kembalikan — batalkan satu aksi simpan.
// Mengembalikan setiap sel ke nilai sebelum aksi tersebut. Bila sebelumnya
// baris belum ada sama sekali, barisnya dihapus lagi (kembali ke kondisi semula).
// Aksi pengembalian ini sendiri dicatat sebagai batch baru, sehingga tetap terlacak.
func (h *Handler) KembalikanBatchSPP(w http.ResponseWriter, r *http.Request) {
	batch := chi.URLParam(r, "batch")

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

	var sudah bool
	var taStart int
	if err := tx.QueryRow(`SELECT dikembalikan, tahun_ajaran FROM spp_riwayat_batch WHERE batch_id = ?`, batch).
		Scan(&sudah, &taStart); err == sql.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Riwayat tidak ditemukan")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if sudah {
		httpx.Error(w, http.StatusConflict, "SUDAH_DIKEMBALIKAN", "Perubahan ini sudah pernah dikembalikan")
		return
	}

	rows, err := tx.Query(`
		SELECT santri_id, tahun, bulan, lunas_lama, lunas_baru, ket_lama, ket_baru
		FROM spp_riwayat WHERE batch_id = ?`, batch)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	type baris struct {
		SantriID         int64
		Tahun, Bulan     int
		LunasLama        sql.NullBool
		LunasBaru        bool
		KetLama, KetBaru sql.NullString
	}
	daftar := []baris{}
	for rows.Next() {
		var b baris
		if err := rows.Scan(&b.SantriID, &b.Tahun, &b.Bulan, &b.LunasLama, &b.LunasBaru,
			&b.KetLama, &b.KetBaru); err == nil {
			daftar = append(daftar, b)
		}
	}
	rows.Close()
	if len(daftar) == 0 {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Tidak ada rincian perubahan untuk dikembalikan")
		return
	}

	batchBaru := newBatchID()
	upsert, err := tx.Prepare(`
		INSERT INTO spp (santri_id, tahun, bulan, lunas, keterangan, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE lunas = VALUES(lunas), keterangan = VALUES(keterangan)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer upsert.Close()

	hapus, err := tx.Prepare(`DELETE FROM spp WHERE santri_id = ? AND tahun = ? AND bulan = ?`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer hapus.Close()

	catat, err := tx.Prepare(`
		INSERT INTO spp_riwayat (batch_id, santri_id, tahun, bulan, lunas_lama, lunas_baru,
		                         ket_lama, ket_baru, aksi, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'kembalikan', ?)`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer catat.Close()

	for _, b := range daftar {
		if b.LunasLama.Valid {
			if _, err := upsert.Exec(b.SantriID, b.Tahun, b.Bulan, b.LunasLama.Bool, b.KetLama, userID); err != nil {
				httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
				return
			}
		} else {
			// sebelumnya baris belum ada → hapus supaya benar-benar kembali seperti semula
			if _, err := hapus.Exec(b.SantriID, b.Tahun, b.Bulan); err != nil {
				httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
				return
			}
		}
		// catat arah kebalikannya: dari nilai_baru kembali ke nilai_lama
		var lunasBaruSetelahKembali bool
		if b.LunasLama.Valid {
			lunasBaruSetelahKembali = b.LunasLama.Bool
		}
		if _, err := catat.Exec(batchBaru, b.SantriID, b.Tahun, b.Bulan,
			b.LunasBaru, lunasBaruSetelahKembali, b.KetBaru, b.KetLama, userID); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO spp_riwayat_batch (batch_id, aksi, jumlah_sel, tahun_ajaran, asal_batch, created_by)
		VALUES (?, 'kembalikan', ?, ?, ?, ?)`,
		batchBaru, len(daftar), taStart, batch, userID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE spp_riwayat_batch SET dikembalikan = 1 WHERE batch_id = ?`, batch); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"dikembalikan": len(daftar), "batch_baru": batchBaru})
}
