package notifworker

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"sim-madrasah/backend/internal/handlers"
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
// Sengaja TIDAK memanggil Telegram langsung: pesan hanya diantrekan ke tabel
// `notifikasi`. Bila pengiriman sedang dijeda dari halaman admin, pengingat tetap
// tercatat dan terlihat di halaman Notifikasi, lalu terkirim sendiri begitu
// pengiriman diaktifkan lagi.
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

	tujuan := chatTujuan(db)
	if tujuan == "" {
		// tetap tandai supaya tidak diperiksa berulang; catat agar admin tahu
		tandaiPengingatTerkirim(db, hariIni)
		log.Printf("pengingat-absensi: tujuan chat Telegram belum diatur — pengingat dilewati")
		return nil
	}

	if _, err := db.Exec(`
		INSERT INTO notifikasi (santri_id, jenis, ref_tanggal, tujuan, pesan, status)
		VALUES (NULL, ?, ?, ?, ?, 'pending')`,
		jenisPengingatAbsensi, hariIni, tujuan, pesan); err != nil {
		return fmt.Errorf("gagal mengantrekan pengingat: %w", err)
	}
	tandaiPengingatTerkirim(db, hariIni)
	log.Printf("pengingat-absensi: %s diantrekan ke %s (%d kelas belum, %d sebagian)",
		hariIni, tujuan, len(belum), len(sebagian))
	return nil
}

// chatTujuan mengambil tujuan pengiriman Telegram. Kosong = belum diatur admin.
// Dipakai bersama oleh pengingat absensi dan pengingat SPP.
func chatTujuan(db *sql.DB) string {
	var chatID sql.NullString
	_ = db.QueryRow(`SELECT chat_id FROM telegram_pengaturan WHERE id = 1`).Scan(&chatID)
	if !chatID.Valid {
		return ""
	}
	return strings.TrimSpace(chatID.String)
}

func tandaiPengingatTerkirim(db *sql.DB, tanggal string) {
	if _, err := db.Exec(`UPDATE pengingat_absensi_pengaturan SET terakhir_kirim = ? WHERE id = 1`, tanggal); err != nil {
		log.Printf("pengingat-absensi: gagal menandai terkirim: %v", err)
	}
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
