package handlers

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"sim-madrasah/backend/internal/httpx"
	"sim-madrasah/backend/internal/waid"
)

const jenisAbsensiAlpha = "absensi_alpha"

var hariIndo = []string{"Ahad", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
var bulanIndo = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func tanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d", hariIndo[int(t.Weekday())], t.Day(), bulanIndo[int(t.Month())-1], t.Year())
}

// pesanAlpha menyusun teks WhatsApp untuk orang tua.
func pesanAlpha(nama, kelas string, tgl time.Time) string {
	return fmt.Sprintf(
		"Assalamu'alaikum warahmatullahi wabarakatuh.\n\n"+
			"Kami informasikan bahwa ananda *%s* (%s) tercatat *TIDAK HADIR tanpa keterangan (Alpha)* pada %s.\n\n"+
			"Mohon konfirmasi dari Bapak/Ibu. Jazakumullahu khairan.\n\n"+
			"_Madrasah Al Fath_",
		nama, kelas, tanggalIndonesia(tgl))
}

// antreNotifAlpha mengantrikan pesan alpha untuk satu santri (dipanggil di dalam
// transaksi penyimpanan absensi). Dilewati diam-diam bila nomor orang tua kosong
// atau tidak valid — absensi tetap tersimpan.
func antreNotifAlpha(tx *sql.Tx, santriID int64, tanggal string) error {
	var nama, kelas string
	var noOrtu sql.NullString
	err := tx.QueryRow(`
		SELECT s.nama, k.nama, s.no_ortu
		FROM santri s JOIN kelas k ON k.id = s.kelas_id
		WHERE s.id = ?`, santriID).Scan(&nama, &kelas, &noOrtu)
	if err != nil {
		return nil // santri tak ditemukan → jangan gagalkan penyimpanan absensi
	}
	if !noOrtu.Valid || noOrtu.String == "" {
		return nil // belum ada nomor orang tua
	}
	tujuan, err := waid.Normalize(noOrtu.String)
	if err != nil {
		return nil // format nomor tidak valid → lewati
	}
	tgl, err := time.Parse("2006-01-02", tanggal)
	if err != nil {
		return nil
	}

	// Bila sudah pernah TERKIRIM untuk tanggal itu, jangan diubah jadi pending lagi.
	_, err = tx.Exec(`
		INSERT INTO notifikasi_wa (santri_id, jenis, ref_tanggal, tujuan, pesan, status)
		VALUES (?, ?, ?, ?, ?, 'pending')
		ON DUPLICATE KEY UPDATE
			tujuan = VALUES(tujuan),
			pesan  = VALUES(pesan),
			status = IF(status = 'terkirim', 'terkirim', 'pending')`,
		santriID, jenisAbsensiAlpha, tanggal, tujuan, pesanAlpha(nama, kelas, tgl))
	return err
}

// batalkanNotifAlpha membatalkan antrean yang belum terkirim ketika status
// absensi dikoreksi menjadi bukan alpha (mis. guru salah tandai).
func batalkanNotifAlpha(tx *sql.Tx, santriID int64, tanggal string) error {
	_, err := tx.Exec(`
		UPDATE notifikasi_wa SET status = 'batal'
		WHERE santri_id = ? AND jenis = ? AND ref_tanggal = ? AND status = 'pending'`,
		santriID, jenisAbsensiAlpha, tanggal)
	return err
}

// ================= ENDPOINT UNTUK BOT =================

func (h *Handler) botAuthorized(r *http.Request) bool {
	secret := h.Cfg.BotSharedSecret
	if secret == "" {
		return false
	}
	kirim := r.Header.Get("X-Bot-Secret")
	if kirim == "" {
		kirim = r.URL.Query().Get("bot_secret")
	}
	return subtle.ConstantTimeCompare([]byte(kirim), []byte(secret)) == 1
}

type notifKeluar struct {
	ID     int64  `json:"id"`
	Tujuan string `json:"tujuan"`
	Pesan  string `json:"pesan"`
	Jenis  string `json:"jenis"`
}

// GET /notifikasi/pending?limit=  (auth: X-Bot-Secret)
// Dipakai bot WhatsApp untuk mengambil pesan yang perlu dikirim.
func (h *Handler) NotifPending(w http.ResponseWriter, r *http.Request) {
	if !h.botAuthorized(r) {
		httpx.Error(w, http.StatusUnauthorized, "BOT_UNAUTHORIZED", "Secret bot tidak valid")
		return
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	rows, err := h.DB.Query(`
		SELECT id, tujuan, pesan, jenis FROM notifikasi_wa
		WHERE status = 'pending' ORDER BY id LIMIT ?`, limit)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()
	out := []notifKeluar{}
	for rows.Next() {
		var n notifKeluar
		_ = rows.Scan(&n.ID, &n.Tujuan, &n.Pesan, &n.Jenis)
		out = append(out, n)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out})
}

type notifStatusReq struct {
	ID      int64  `json:"id"`
	Sukses  bool   `json:"sukses"`
	Catatan string `json:"catatan"`
}

// POST /notifikasi/status  (auth: X-Bot-Secret)
// Bot melapor hasil pengiriman satu pesan.
func (h *Handler) NotifStatus(w http.ResponseWriter, r *http.Request) {
	if !h.botAuthorized(r) {
		httpx.Error(w, http.StatusUnauthorized, "BOT_UNAUTHORIZED", "Secret bot tidak valid")
		return
	}
	var req notifStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "id wajib")
		return
	}
	var err error
	if req.Sukses {
		_, err = h.DB.Exec(`
			UPDATE notifikasi_wa
			   SET status='terkirim', dikirim_at=NOW(), percobaan=percobaan+1, catatan=NULL
			 WHERE id=?`, req.ID)
	} else {
		catatan := req.Catatan
		if len(catatan) > 255 {
			catatan = catatan[:255]
		}
		_, err = h.DB.Exec(`
			UPDATE notifikasi_wa
			   SET status='gagal', percobaan=percobaan+1, catatan=?
			 WHERE id=?`, catatan, req.ID)
	}
	if err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// ================= ENDPOINT ADMIN =================

// GET /notifikasi?status=&limit=
func (h *Handler) ListNotifikasi(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	q := `SELECT n.id, COALESCE(s.nama,'-'), n.jenis, n.ref_tanggal, n.tujuan, n.pesan,
	             n.status, n.percobaan, n.catatan, n.dikirim_at, n.created_at
	      FROM notifikasi_wa n LEFT JOIN santri s ON s.id = n.santri_id`
	args := []interface{}{}
	if status != "" {
		q += ` WHERE n.status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY n.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := h.DB.Query(q, args...)
	if err != nil {
		dbErr(w, err)
		return
	}
	defer rows.Close()

	type item struct {
		ID         int64   `json:"id"`
		Nama       string  `json:"nama"`
		Jenis      string  `json:"jenis"`
		RefTanggal *string `json:"ref_tanggal"`
		Tujuan     string  `json:"tujuan"`
		Pesan      string  `json:"pesan"`
		Status     string  `json:"status"`
		Percobaan  int     `json:"percobaan"`
		Catatan    *string `json:"catatan"`
		DikirimAt  *string `json:"dikirim_at"`
		CreatedAt  string  `json:"created_at"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var ref, kirim sql.NullString
		var created string
		_ = rows.Scan(&it.ID, &it.Nama, &it.Jenis, &ref, &it.Tujuan, &it.Pesan,
			&it.Status, &it.Percobaan, &it.Catatan, &kirim, &created)
		if ref.Valid {
			t := ref.String
			if len(t) > 10 {
				t = t[:10]
			}
			it.RefTanggal = &t
		}
		if kirim.Valid {
			it.DikirimAt = &kirim.String
		}
		it.CreatedAt = created
		out = append(out, it)
	}

	// ringkasan jumlah per status
	ringkas := map[string]int{}
	srows, err := h.DB.Query(`SELECT status, COUNT(*) FROM notifikasi_wa GROUP BY status`)
	if err == nil {
		for srows.Next() {
			var s string
			var c int
			_ = srows.Scan(&s, &c)
			ringkas[s] = c
		}
		srows.Close()
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": out, "ringkasan": ringkas})
}

// POST /notifikasi/{id}/ulang — kembalikan ke antrean (untuk yang gagal/batal).
func (h *Handler) UlangNotifikasi(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`UPDATE notifikasi_wa SET status='pending', catatan=NULL WHERE id=?`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// POST /notifikasi/{id}/batal — batalkan pesan yang belum terkirim.
func (h *Handler) BatalNotifikasi(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.DB.Exec(`UPDATE notifikasi_wa SET status='batal' WHERE id=? AND status <> 'terkirim'`, id); err != nil {
		dbErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}
