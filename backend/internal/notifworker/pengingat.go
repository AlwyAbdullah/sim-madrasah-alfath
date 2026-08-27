package notifworker

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"sim-madrasah/backend/internal/handlers"
)

// RunPengingatAbsensi memeriksa tiap menit apakah sudah waktunya mengirim
// pengingat "masih ada kelas yang belum diabsen".
//
// Berkas ini hanya mengurus JADWAL. Penyusunan dan penyaluran pesannya ada di
// handlers.KirimPengingatAbsensi, supaya logika yang sama dipakai oleh tombol
// "Kirim sekarang" di halaman admin — kalau dipisah, keduanya pasti menyimpang.
//
// Pesan hanya diantrekan ke tabel `notifikasi`; bila pengiriman sedang dijeda
// dari halaman admin, pengingat tetap tercatat dan terkirim begitu diaktifkan.
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
	if terakhir.Valid && strings.HasPrefix(terakhir.String, hariIni) {
		return nil // sudah dikirim hari ini
	}
	// Memakai perbandingan ">=" (bukan "== menit ini") supaya pengingat tetap
	// terkirim walau server sempat mati saat jam tersebut.
	if now.Hour() < jam || (now.Hour() == jam && now.Minute() < menit) {
		return nil
	}
	if !handlers.HariSekolah(db, now) {
		return nil // Kamis/Jumat atau hari libur — tidak perlu diingatkan
	}

	hasil, err := handlers.KirimPengingatAbsensi(db, now, hariIni)
	if err != nil {
		return err
	}
	tandaiPengingatTerkirim(db, hariIni)

	if hasil.KelasPerluDiisi == 0 {
		log.Printf("pengingat-absensi: %s semua kelas sudah lengkap — tidak ada pengingat", hariIni)
		return nil
	}
	log.Printf("pengingat-absensi: %s — %d kelas perlu diisi; %d pesan ke wali, %d ke superadmin, %d kelas ke grup",
		hariIni, hasil.KelasPerluDiisi, hasil.KeWali, hasil.KeSuperadmin, hasil.KelasKeGrup)
	return nil
}

func tandaiPengingatTerkirim(db *sql.DB, tanggal string) {
	if _, err := db.Exec(`UPDATE pengingat_absensi_pengaturan SET terakhir_kirim = ? WHERE id = 1`, tanggal); err != nil {
		log.Printf("pengingat-absensi: gagal menandai terkirim: %v", err)
	}
}
