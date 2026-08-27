package notifworker

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"sim-madrasah/backend/internal/handlers"
)

// RunPengingatSPP memeriksa tiap menit apakah sudah waktunya mengirim rekap
// "santri yang belum membayar SPP bulan ini".
//
// Sama seperti pengingat absensi, pesan hanya DIANTREKAN ke tabel `notifikasi`;
// worker Telegram yang mengirimnya. Jadi bila pengiriman sedang dijeda dari
// halaman admin, pengingat tetap tercatat dan terkirim begitu diaktifkan lagi.
func RunPengingatSPP(db *sql.DB) {
	log.Println("pengingat-spp: aktif — memeriksa tiap menit")
	for {
		if err := periksaDanKirimSPP(db, time.Now()); err != nil {
			log.Printf("pengingat-spp: %v", err)
		}
		time.Sleep(time.Minute)
	}
}

func periksaDanKirimSPP(db *sql.DB, now time.Time) error {
	var aktif bool
	var tanggal, jam, menit int
	var terakhir sql.NullString
	err := db.QueryRow(`
		SELECT aktif, tanggal, jam, menit, terakhir_kirim FROM pengingat_spp_pengaturan WHERE id = 1`).
		Scan(&aktif, &tanggal, &jam, &menit, &terakhir)
	if err != nil {
		return nil // tabel belum ada (migrasi belum jalan) — diamkan saja
	}
	if !aktif {
		return nil
	}

	bulanIni := now.Format("2006-01")
	// sudah dikirim bulan ini?
	if terakhir.Valid && terakhir.String == bulanIni {
		return nil
	}
	// belum waktunya. Perbandingan ">=" (bukan "== tepat menit ini") supaya
	// pengingat tetap terkirim walau server sempat mati pada jam tersebut —
	// termasuk bila baru menyala beberapa hari setelah tanggal yang dipilih.
	if now.Day() < tanggal {
		return nil
	}
	if now.Day() == tanggal && (now.Hour() < jam || (now.Hour() == jam && now.Minute() < menit)) {
		return nil
	}

	rekap, err := handlers.HitungTunggakanSPP(db, now.Year(), int(now.Month()))
	if err != nil {
		return fmt.Errorf("gagal menghitung tunggakan: %w", err)
	}

	// semua sudah lunas → tidak perlu mengganggu, tapi tetap ditandai agar tidak
	// diperiksa berulang sepanjang sisa bulan
	if rekap.Belum == 0 {
		tandaiPengingatSPPTerkirim(db, bulanIni)
		log.Printf("pengingat-spp: %s seluruh santri sudah lunas — tidak ada pengingat", bulanIni)
		return nil
	}

	tujuan := handlers.ChatGrup(db)
	if tujuan == "" {
		tandaiPengingatSPPTerkirim(db, bulanIni)
		log.Printf("pengingat-spp: tujuan chat Telegram belum diatur — pengingat dilewati")
		return nil
	}

	ref := fmt.Sprintf("%s-01", bulanIni)
	if err := handlers.AntreNotifikasi(db, handlers.JenisPengingatSPP, ref, tujuan,
		handlers.PesanTunggakanSPP(rekap)); err != nil {
		return fmt.Errorf("gagal mengantrekan pengingat: %w", err)
	}
	tandaiPengingatSPPTerkirim(db, bulanIni)
	log.Printf("pengingat-spp: %s diantrekan ke %s (%d dari %d santri belum bayar)",
		bulanIni, tujuan, rekap.Belum, rekap.TotalSantri)
	return nil
}

func tandaiPengingatSPPTerkirim(db *sql.DB, bulan string) {
	if _, err := db.Exec(`UPDATE pengingat_spp_pengaturan SET terakhir_kirim = ? WHERE id = 1`, bulan); err != nil {
		log.Printf("pengingat-spp: gagal menandai terkirim: %v", err)
	}
}
