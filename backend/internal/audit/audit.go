// Package audit mencatat perubahan yang terjadi di sistem ke tabel log_aktivitas.
//
// Dipisahkan jadi paket sendiri supaya bentuk ringkasannya seragam. Kalau tiap
// handler menyusun sendiri kalimatnya, lini masa akan campur aduk dan sulit
// dibaca — padahal justru keterbacaan itu satu-satunya alasan tabel ini ada.
//
// Aturan penting: DICATAT PER AKSI, BUKAN PER BARIS. Menyimpan absensi satu
// kelas menyentuh belasan baris tapi hanya boleh menghasilkan satu entri.
// Kalau per baris, lognya puluhan entri per hari hanya dari absensi dan tidak
// akan ada yang membacanya.
package audit

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"sim-madrasah/backend/internal/middleware"
)

// Aksi yang dikenal. Dipakai sebagai konstanta agar tidak ada salah ketik yang
// diam-diam memecah penyaringan di halaman Aktivitas.
const (
	Login            = "login"
	LoginGagal       = "login_gagal"
	LoginTerkunci    = "login_terkunci"
	BukaBlokir       = "buka_blokir_login"
	Logout           = "logout"
	PutusSesi        = "putus_sesi"
	SimpanAbsensi    = "simpan_absensi"
	SimpanNilai      = "simpan_nilai"
	SimpanTugas      = "simpan_tugas"
	SimpanSPP        = "simpan_spp"
	KembalikanSPP    = "kembalikan_spp"
	TambahSantri     = "tambah_santri"
	UbahSantri       = "ubah_santri"
	HapusSantri      = "hapus_santri"
	NaikKelas        = "naik_kelas"
	UbahMaster       = "ubah_master"
	BuatAkun         = "buat_akun"
	UbahAkun         = "ubah_akun"
	HapusAkun        = "hapus_akun"
	ResetPassword    = "reset_password"
	GantiPassword    = "ganti_password"
	UbahWaliKelas    = "ubah_wali_kelas"
	UbahPengaturan   = "ubah_pengaturan"
)

// Entri adalah satu baris lini masa.
type Entri struct {
	Aksi      string
	Entitas   string      // mis. "absensi", "santri" — boleh kosong
	EntitasID string      // mis. "3" atau "2026-08-26" — boleh kosong
	Ringkasan string      // kalimat siap baca; WAJIB
	Rincian   interface{} // opsional, disimpan sebagai JSON
}

const maksRingkasan = 500

// Catat menulis satu entri. Sengaja TIDAK mengembalikan error: pencatatan audit
// tidak boleh menggagalkan pekerjaan yang sudah berhasil disimpan. Kegagalan
// ditulis ke log server supaya tetap ketahuan.
func Catat(db *sql.DB, r *http.Request, e Entri) {
	if e.Ringkasan == "" {
		return
	}
	if len(e.Ringkasan) > maksRingkasan {
		e.Ringkasan = e.Ringkasan[:maksRingkasan]
	}

	var userID interface{}
	username, nama := "-", "-"
	if c := middleware.ClaimsFrom(r); c != nil {
		userID = c.UserID
		username = c.Username
		nama = c.Username // diganti nama asli di bawah bila ketemu
		var n string
		if err := db.QueryRow(`SELECT nama FROM users WHERE id = ?`, c.UserID).Scan(&n); err == nil && n != "" {
			nama = n
		}
	}

	var rincian interface{}
	if e.Rincian != nil {
		if b, err := json.Marshal(e.Rincian); err == nil {
			rincian = string(b)
		}
	}

	var entitas, entitasID, ip interface{}
	if e.Entitas != "" {
		entitas = e.Entitas
	}
	if e.EntitasID != "" {
		entitasID = e.EntitasID
	}
	if v := middleware.ClientIP(r); v != "" {
		ip = v
	}

	if _, err := db.Exec(`
		INSERT INTO log_aktivitas (user_id, username, nama, aksi, entitas, entitas_id, ringkasan, rincian, ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, username, nama, e.Aksi, entitas, entitasID, e.Ringkasan, rincian, ip); err != nil {
		log.Printf("audit: gagal mencatat %q: %v", e.Aksi, err)
	}
}

// CatatUser dipakai saat pelakunya belum/tidak ada di context permintaan —
// terutama pencatatan login, yang terjadi sebelum middleware auth berjalan.
func CatatUser(db *sql.DB, r *http.Request, userID int64, username, nama string, e Entri) {
	if e.Ringkasan == "" {
		return
	}
	if len(e.Ringkasan) > maksRingkasan {
		e.Ringkasan = e.Ringkasan[:maksRingkasan]
	}
	var uid interface{}
	if userID > 0 {
		uid = userID
	}
	var ip interface{}
	if v := middleware.ClientIP(r); v != "" {
		ip = v
	}
	var rincian interface{}
	if e.Rincian != nil {
		if b, err := json.Marshal(e.Rincian); err == nil {
			rincian = string(b)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO log_aktivitas (user_id, username, nama, aksi, entitas, entitas_id, ringkasan, rincian, ip)
		VALUES (?, ?, ?, ?, NULL, NULL, ?, ?, ?)`,
		uid, username, nama, e.Aksi, e.Ringkasan, rincian, ip); err != nil {
		log.Printf("audit: gagal mencatat %q: %v", e.Aksi, err)
	}
}
