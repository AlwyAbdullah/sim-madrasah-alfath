package notifworker

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"sim-madrasah/backend/internal/handlers"
	"sim-madrasah/backend/internal/waid"
)

const jenisPengingatAbsensi = "pengingat_absensi"

var hariIndo = []string{"Ahad", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
var bulanIndo = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func tanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d", hariIndo[int(t.Weekday())], t.Day(), bulanIndo[int(t.Month())-1], t.Year())
}

// RunPengingatAbsensi memeriksa tiap menit apakah sudah waktunya mengirim
// pengingat "masih ada kelas yang belum diabsen".
//
// Sengaja TIDAK bergantung pada WAHA: pesan hanya diantrekan ke tabel
// notifikasi_wa. Bila WAHA sedang mati, pengingat tetap tercatat dan terlihat di
// halaman Notifikasi WA, lalu terkirim sendiri begitu WAHA dinyalakan.
func RunPengingatAbsensi(db *sql.DB) {
	log.Println("pengingat-absensi: aktif — memeriksa tiap menit")
	for {
		if err := periksaDanKirim(db, time.Now()); err != nil {
			log.Printf("pengingat-absensi: %v", err)
		}
		time.Sleep(time.Minute)
	}
}

func periksaDanKirim(db *sql.DB, now time.Time) error {
	var aktif bool
	var jam, menit int
	var terakhir sql.NullString
	err := db.QueryRow(`
		SELECT aktif, jam, menit, terakhir_kirim FROM pengingat_absensi_pengaturan WHERE id = 1`).
		Scan(&aktif, &jam, &menit, &terakhir)
	if err != nil {
		return nil // tabel belum ada (migrasi belum jalan) — diamkan saja
	}
	if !aktif {
		return nil
	}

	hariIni := now.Format("2006-01-02")
	// sudah pernah dikirim hari ini?
	if terakhir.Valid && strings.HasPrefix(terakhir.String, hariIni) {
		return nil
	}
	// belum waktunya. Memakai perbandingan ">=" (bukan "== menit ini") supaya
	// pengingat tetap terkirim walau server sempat mati saat jam tersebut.
	if now.Hour() < jam || (now.Hour() == jam && now.Minute() < menit) {
		return nil
	}
	if !handlers.HariSekolah(db, now) {
		return nil // Kamis/Jumat atau hari libur — tidak perlu diingatkan
	}

	daftar, err := handlers.StatusAbsensiHarian(db, hariIni)
	if err != nil {
		return fmt.Errorf("gagal memeriksa absensi: %w", err)
	}

	var belum, sebagian []handlers.StatusKelasAbsensi
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

	// semua sudah lengkap → tidak perlu mengganggu, tapi tetap tandai agar
	// tidak diperiksa berulang sepanjang malam
	if len(belum) == 0 && len(sebagian) == 0 {
		tandaiPengingatTerkirim(db, hariIni)
		log.Printf("pengingat-absensi: %s semua kelas sudah lengkap — tidak ada pengingat", hariIni)
		return nil
	}

	pesan := susunPesan(now, belum, sebagian, lengkap)

	// Utamakan Telegram (API resmi, tanpa risiko blokir). Bila chat Telegram belum
	// diatur, jatuh kembali ke WhatsApp memakai nomor admin.
	kanal, tujuan := tujuanPengingat(db)
	if len(tujuan) == 0 {
		// tetap tandai supaya tidak berulang; tapi catat agar admin tahu
		tandaiPengingatTerkirim(db, hariIni)
		log.Printf("pengingat-absensi: belum ada tujuan (chat Telegram / nomor WA admin) — pengingat dilewati")
		return nil
	}

	for _, t := range tujuan {
		if _, err := db.Exec(`
			INSERT INTO notifikasi_wa (santri_id, jenis, kanal, ref_tanggal, tujuan, pesan, status)
			VALUES (NULL, ?, ?, ?, ?, ?, 'pending')`,
			jenisPengingatAbsensi, kanal, hariIni, t, pesan); err != nil {
			return fmt.Errorf("gagal mengantrekan pengingat: %w", err)
		}
	}
	tandaiPengingatTerkirim(db, hariIni)
	log.Printf("pengingat-absensi: %s diantrekan lewat %s ke %d tujuan (%d kelas belum, %d sebagian)",
		hariIni, kanal, len(tujuan), len(belum), len(sebagian))
	return nil
}

// tujuanPengingat memilih kanal & daftar tujuan pengiriman.
func tujuanPengingat(db *sql.DB) (string, []string) {
	var chatID sql.NullString
	_ = db.QueryRow(`SELECT chat_id FROM telegram_pengaturan WHERE id = 1`).Scan(&chatID)
	if chatID.Valid && strings.TrimSpace(chatID.String) != "" {
		return "telegram", []string{strings.TrimSpace(chatID.String)}
	}
	nomor, err := nomorAdmin(db)
	if err != nil {
		return "whatsapp", nil
	}
	return "whatsapp", nomor
}

func tandaiPengingatTerkirim(db *sql.DB, tanggal string) {
	if _, err := db.Exec(`UPDATE pengingat_absensi_pengaturan SET terakhir_kirim = ? WHERE id = 1`, tanggal); err != nil {
		log.Printf("pengingat-absensi: gagal menandai terkirim: %v", err)
	}
}

// nomorAdmin mengambil nomor WhatsApp seluruh admin aktif yang sudah mengisinya.
func nomorAdmin(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT whatsapp_number FROM users
		WHERE role = 'admin' AND is_active = 1
		  AND whatsapp_number IS NOT NULL AND whatsapp_number <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var no string
		if err := rows.Scan(&no); err != nil {
			continue
		}
		if wa, err := waid.Normalize(no); err == nil {
			out = append(out, wa)
		}
	}
	return out, rows.Err()
}

func susunPesan(now time.Time, belum, sebagian []handlers.StatusKelasAbsensi, lengkap int) string {
	var b strings.Builder
	b.WriteString("Assalamu'alaikum warahmatullahi wabarakatuh.\n\n")
	b.WriteString(fmt.Sprintf("Pengingat absensi *%s*.\n", tanggalIndonesia(now)))

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
