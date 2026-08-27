package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/httpx"
)

// JenisPengingatAbsensi menandai baris antrean yang berasal dari pengingat absensi.
const JenisPengingatAbsensi = "pengingat_absensi"

var hariIndo = []string{"Ahad", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

// TanggalIndonesia mis. "Rabu, 26 Agustus 2026".
func TanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d", hariIndo[int(t.Weekday())], t.Day(), bulanIndo[int(t.Month())-1], t.Year())
}

// HasilPengingat merangkum ke mana saja pengingat dikirim.
type HasilPengingat struct {
	KelasPerluDiisi int `json:"kelas_perlu_diisi"`
	KeWali          int `json:"ke_wali"`
	KeSuperadmin    int `json:"ke_superadmin"`
	KelasKeGrup     int `json:"kelas_ke_grup"`
}

// KirimPengingatAbsensi menyusun dan mengantrekan pengingat untuk satu tanggal.
//
// Prinsip penyalurannya:
//  1. wali kelas yang sudah menautkan Telegram menerima kabar KELASNYA SENDIRI —
//     pengingat yang menyebut nama jauh lebih mungkin ditindaklanjuti daripada
//     "Kelas 3 belum diabsen" yang dibaca 20 orang di grup;
//  2. superadmin selalu menerima ringkasan seluruh madrasah;
//  3. kelas yang walinya belum menautkan Telegram tetap dikabarkan ke grup,
//     supaya tidak ada kelas yang luput hanya karena walinya belum siap.
//
// Pesan hanya DIANTREKAN; worker Telegram yang mengirimnya.
func KirimPengingatAbsensi(db *sql.DB, now time.Time, tanggal string) (HasilPengingat, error) {
	var hasil HasilPengingat

	daftar, err := StatusAbsensiHarian(db, tanggal)
	if err != nil {
		return hasil, fmt.Errorf("gagal memeriksa absensi: %w", err)
	}

	var belum, sebagian []StatusKelasAbsensi
	lengkap := 0
	for _, k := range daftar {
		switch k.Status {
		case "belum":
			belum = append(belum, k)
		case "sebagian":
			sebagian = append(sebagian, k)
		default:
			lengkap++
		}
	}
	perlu := append(append([]StatusKelasAbsensi{}, belum...), sebagian...)
	hasil.KelasPerluDiisi = len(perlu)
	if len(perlu) == 0 {
		return hasil, nil
	}

	tertangani, keWali := kirimKeWali(db, now, tanggal, perlu)
	hasil.KeWali = keWali
	hasil.KeSuperadmin = kirimKeSuperadmin(db, now, tanggal, belum, sebagian, lengkap)

	var sisaBelum, sisaSebagian []StatusKelasAbsensi
	for _, k := range perlu {
		if tertangani[k.KelasID] {
			continue
		}
		if k.Status == "belum" {
			sisaBelum = append(sisaBelum, k)
		} else {
			sisaSebagian = append(sisaSebagian, k)
		}
	}
	sisa := len(sisaBelum) + len(sisaSebagian)
	if sisa > 0 {
		tujuan := ChatGrup(db)
		if tujuan == "" {
			log.Printf("pengingat-absensi: %d kelas tanpa wali ber-Telegram, dan chat grup belum diatur", sisa)
		} else if err := AntreNotifikasi(db, JenisPengingatAbsensi, tanggal, tujuan,
			susunPesanRingkasan(now, sisaBelum, sisaSebagian, lengkap)); err != nil {
			return hasil, err
		} else {
			hasil.KelasKeGrup = sisa
		}
	}
	return hasil, nil
}

// ChatGrup mengambil tujuan grup Telegram. Kosong = belum diatur admin.
// Dipakai bersama oleh pengingat absensi dan pengingat SPP.
func ChatGrup(db *sql.DB) string {
	var chatID sql.NullString
	_ = db.QueryRow(`SELECT chat_id FROM telegram_pengaturan WHERE id = 1`).Scan(&chatID)
	return strings.TrimSpace(chatID.String)
}

// kirimKeWali mengelompokkan per orang: wali dua kelas menerima SATU pesan
// berisi kedua kelasnya, bukan dua pesan terpisah.
func kirimKeWali(db *sql.DB, now time.Time, tanggal string,
	perlu []StatusKelasAbsensi) (map[int64]bool, int) {

	tertangani := map[int64]bool{}
	perWali := map[int64][]StatusKelasAbsensi{}
	nama := map[int64]string{}
	chat := map[int64]string{}

	for _, k := range perlu {
		rows, err := db.Query(`
			SELECT u.id, u.nama, u.telegram_user_id
			FROM kelas_wali kw JOIN users u ON u.id = kw.user_id
			WHERE kw.kelas_id = ? AND u.is_active = 1 AND u.telegram_user_id IS NOT NULL`, k.KelasID)
		if err != nil {
			log.Printf("pengingat-absensi: gagal mencari wali kelas %d: %v", k.KelasID, err)
			continue
		}
		for rows.Next() {
			var uid, chatID int64
			var n string
			if rows.Scan(&uid, &n, &chatID) != nil {
				continue
			}
			perWali[uid] = append(perWali[uid], k)
			nama[uid] = strings.TrimSpace(n)
			chat[uid] = fmt.Sprintf("%d", chatID)
			tertangani[k.KelasID] = true
		}
		rows.Close()
	}

	terkirim := 0
	for uid, kelas := range perWali {
		if err := AntreNotifikasi(db, JenisPengingatAbsensi, tanggal, chat[uid],
			pesanWali(now, nama[uid], kelas)); err != nil {
			log.Printf("pengingat-absensi: %v", err)
			continue
		}
		terkirim++
	}
	return tertangani, terkirim
}

func kirimKeSuperadmin(db *sql.DB, now time.Time, tanggal string,
	belum, sebagian []StatusKelasAbsensi, lengkap int) int {

	rows, err := db.Query(`
		SELECT telegram_user_id FROM users
		WHERE role = 'superadmin' AND is_active = 1 AND telegram_user_id IS NOT NULL`)
	if err != nil {
		log.Printf("pengingat-absensi: gagal mencari superadmin: %v", err)
		return 0
	}
	defer rows.Close()

	pesan := susunPesanRingkasan(now, belum, sebagian, lengkap)
	terkirim := 0
	for rows.Next() {
		var chatID int64
		if rows.Scan(&chatID) != nil {
			continue
		}
		if err := AntreNotifikasi(db, JenisPengingatAbsensi, tanggal, fmt.Sprintf("%d", chatID), pesan); err != nil {
			log.Printf("pengingat-absensi: %v", err)
			continue
		}
		terkirim++
	}
	return terkirim
}

func pesanWali(now time.Time, nama string, kelas []StatusKelasAbsensi) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Assalamu'alaikum, *%s*.\n\n", nama))
	if len(kelas) == 1 {
		k := kelas[0]
		if k.Status == "belum" {
			b.WriteString(fmt.Sprintf("Absensi *%s* hari ini belum diisi (%d santri).\n", k.Kelas, k.Santri))
		} else {
			b.WriteString(fmt.Sprintf("Absensi *%s* baru terisi %d dari %d santri.\n", k.Kelas, k.Terisi, k.Santri))
		}
	} else {
		b.WriteString("Kelas yang Anda ampu belum lengkap absensinya:\n")
		for _, k := range kelas {
			if k.Status == "belum" {
				b.WriteString(fmt.Sprintf("• %s — belum diisi (%d santri)\n", k.Kelas, k.Santri))
			} else {
				b.WriteString(fmt.Sprintf("• %s — %d dari %d santri\n", k.Kelas, k.Terisi, k.Santri))
			}
		}
	}
	b.WriteString(fmt.Sprintf("\n_%s_\n", TanggalIndonesia(now)))
	b.WriteString("\nMohon dilengkapi. Jazakumullahu khairan.\n\n_SIM Madrasah Al Fath_")
	return b.String()
}

func susunPesanRingkasan(now time.Time, belum, sebagian []StatusKelasAbsensi, lengkap int) string {
	var b strings.Builder
	b.WriteString("Assalamu'alaikum warahmatullahi wabarakatuh.\n\n")
	b.WriteString(fmt.Sprintf("Pengingat absensi *%s*.\n", TanggalIndonesia(now)))

	if len(belum) > 0 {
		b.WriteString(fmt.Sprintf("\n*Belum diabsen (%d kelas):*\n", len(belum)))
		for _, k := range belum {
			b.WriteString(fmt.Sprintf("• %s (%d santri)\n", k.Kelas, k.Santri))
		}
	}
	if len(sebagian) > 0 {
		b.WriteString(fmt.Sprintf("\n*Baru sebagian (%d kelas):*\n", len(sebagian)))
		for _, k := range sebagian {
			b.WriteString(fmt.Sprintf("• %s — %d dari %d santri\n", k.Kelas, k.Terisi, k.Santri))
		}
	}
	if lengkap > 0 {
		b.WriteString(fmt.Sprintf("\nSudah lengkap: %d kelas.\n", lengkap))
	}
	b.WriteString("\nMohon dilengkapi. Jazakumullahu khairan.\n\n_SIM Madrasah Al Fath_")
	return b.String()
}

// POST /pengingat-absensi/kirim?tanggal=YYYY-MM-DD
//
// Kirim di luar jadwal. Sengaja TIDAK mengubah `terakhir_kirim`, agar pengiriman
// manual tidak membatalkan pengingat terjadwal hari itu — prinsip yang sama
// dengan tombol "Kirim sekarang" pada pengingat SPP.
func (h *Handler) KirimPengingatAbsensiSekarang(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	if v := r.URL.Query().Get("tanggal"); v != "" {
		p, err := time.Parse("2006-01-02", v)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal harus YYYY-MM-DD")
			return
		}
		t = p
	}
	tanggal := t.Format("2006-01-02")

	hasil, err := KirimPengingatAbsensi(h.DB, t, tanggal)
	if err != nil {
		dbErr(w, err)
		return
	}
	if hasil.KelasPerluDiisi == 0 {
		httpx.JSON(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("Semua kelas sudah lengkap absensinya pada %s — tidak ada yang perlu diingatkan.", tanggal),
			"hasil":   hasil,
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Pengingat diantrekan: %d pesan ke wali kelas, %d ke superadmin, %d kelas dikabarkan ke grup.",
			hasil.KeWali, hasil.KeSuperadmin, hasil.KelasKeGrup),
		"hasil": hasil,
	})
}
