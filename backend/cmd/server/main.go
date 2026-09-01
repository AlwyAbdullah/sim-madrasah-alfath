package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"sim-madrasah/backend/internal/config"
	"sim-madrasah/backend/internal/db"
	"sim-madrasah/backend/internal/handlers"
	"sim-madrasah/backend/internal/middleware"
	"sim-madrasah/backend/internal/notifworker"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("gagal koneksi DB: %v", err)
	}
	defer conn.Close()

	h := handlers.New(conn, cfg)

	// Pengirim notifikasi — Telegram (API resmi, satu-satunya kanal).
	go notifworker.RunTelegram(conn, cfg)
	// Pembaca pesan masuk — hanya untuk menautkan akun ke chat Telegram.
	go notifworker.RunTelegramMasuk(conn, cfg)
	// Pengingat harian bila masih ada kelas yang belum diabsen.
	go notifworker.RunPengingatAbsensi(conn)
	// Pengingat bulanan daftar santri yang belum membayar SPP.
	go notifworker.RunPengingatSPP(conn)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CorsOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Publik. Pembatasan percobaan login ada DI DALAM handler (lihat
		// internal/gembok): yang dikunci nama akun, dan nama akun baru
		// diketahui setelah badan permintaan dibaca.
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		// Terproteksi (semua guru bisa akses semua kelas — tanpa batasan kelas ampu)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWTSecret, conn))

			r.Get("/auth/me", h.Me)
			// ganti password sendiri — semua peran
			r.Post("/auth/ganti-password", h.GantiPasswordSaya)

			// menautkan akun sendiri ke Telegram — semua peran, terutama wali kelas
			r.Get("/telegram/tautan", h.StatusTautanTelegram)
			r.Post("/telegram/tautan", h.BuatKodeTautanTelegram)
			r.Delete("/telegram/tautan", h.HapusTautanTelegram)

			// Master (baca — semua role login)
			r.Get("/kelas", h.ListKelas)
			r.Get("/kelas/{id}/mapel", h.ListKelasMapel)
			r.Get("/mata-pelajaran", h.ListMapel)
			r.Get("/periode", h.ListPeriode)
			r.Get("/santri", h.ListSantri)

			// Dashboard
			r.Get("/dashboard/summary", h.Summary)
			// status pengisian absensi hari ini (banner Dashboard — guru juga perlu melihat)
			r.Get("/pengingat-absensi", h.GetPengingatAbsensi)
			r.Get("/santri/{id}/detail", h.SantriDetail)

			// Rapor
			r.Get("/rapor", h.RaporData)

			// Absensi
			r.Get("/absensi", h.GetAbsensi)
			r.Post("/absensi/batch", h.SaveAbsensi)
			r.Get("/absensi/export", h.ExportAbsensi)
			r.Get("/absensi/rekap", h.RekapAbsensi)
			r.Get("/absensi/rekap/export", h.ExportRekapAbsensi)

			// Nilai
			r.Get("/nilai", h.GetNilai)
			r.Post("/nilai/batch", h.SaveNilai)
			r.Get("/nilai/export", h.ExportNilai)
			r.Get("/nilai/leger", h.LegerNilai)
			r.Get("/nilai/leger/export", h.ExportLeger)
			r.Get("/nilai/tugas", h.GetTugas)
			r.Post("/nilai/tugas/batch", h.SaveTugasBatch)

			// Catatan (Fase B)
			r.Get("/catatan", h.GetCatatan)
			r.Post("/catatan", h.CreateCatatan)

			// Tugas/PR (Fase B)
			r.Get("/tugas", h.GetTugasList)
			r.Post("/tugas", h.CreateTugas)

			// ===== KHUSUS ADMIN =====
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))

				// SPP (tidak boleh dilihat guru)
				r.Get("/spp", h.GetSPP)
				r.Post("/spp/toggle", h.ToggleSPP)
				r.Post("/spp/batch", h.SaveSPPBatch)
				r.Get("/spp/export", h.ExportSPP)
				r.Get("/spp/riwayat", h.ListRiwayatSPP)
				r.Get("/spp/riwayat/{batch}/detail", h.DetailRiwayatSPP)
				r.Post("/spp/riwayat/{batch}/kembalikan", h.KembalikanBatchSPP)

				// Master CRUD
				r.Post("/kelas", h.CreateKelas)
				r.Put("/kelas/{id}", h.UpdateKelas)
				r.Delete("/kelas/{id}", h.DeleteKelas)
				r.Put("/kelas/{id}/mapel", h.SetKelasMapel)

				r.Post("/santri", h.CreateSantri)
				r.Put("/santri/{id}", h.UpdateSantri)
				r.Delete("/santri/{id}", h.DeleteSantri)
				r.Post("/santri/import", h.ImportSantri)
				r.Post("/santri/naik-kelas", h.NaikKelas)

				r.Post("/mata-pelajaran", h.CreateMapel)
				r.Put("/mata-pelajaran/{id}", h.UpdateMapel)
				r.Delete("/mata-pelajaran/{id}", h.DeleteMapel)

				r.Post("/periode", h.CreatePeriode)
				r.Put("/periode/{id}", h.UpdatePeriode)
				r.Delete("/periode/{id}", h.DeletePeriode)

				r.Get("/hari-libur", h.ListLibur)
				r.Post("/hari-libur", h.CreateLibur)
				r.Delete("/hari-libur/{id}", h.DeleteLibur)

				r.Get("/users", h.ListUsers)
				r.Post("/users", h.CreateUser)
				r.Put("/users/{id}", h.UpdateUser)
				r.Delete("/users/{id}", h.DeleteUser)
				// akun massal dari master guru (idempoten — guru yang sudah punya dilewati)
				r.Get("/users/dari-guru", h.PreviewAkunGuru)
				r.Post("/users/dari-guru", h.BuatAkunDariGuru)

				// wali kelas (boleh lebih dari satu per kelas)
				r.Get("/kelas/{id}/wali", h.GetWaliKelas)
				r.Put("/kelas/{id}/wali", h.SetWaliKelas)

				// lini masa perubahan sistem
				r.Get("/aktivitas", h.ListAktivitas)
				// sesi yang sedang berjalan
				r.Get("/sesi", h.ListSesi)
				// akun/alamat yang sedang terkunci karena salah password berkali-kali
				r.Get("/login-blokir", h.ListBlokirLogin)
				r.Post("/login-blokir/buka", h.BukaBlokirLogin)

				r.Get("/guru", h.ListGuru)
				r.Post("/guru", h.CreateGuru)
				r.Put("/guru/{id}", h.UpdateGuru)
				r.Delete("/guru/{id}", h.DeleteGuru)

				// Notifikasi (pantau & kelola antrean)
				r.Get("/notifikasi", h.ListNotifikasi)
				r.Post("/notifikasi/{id}/ulang", h.UlangNotifikasi)
				r.Post("/notifikasi/{id}/batal", h.BatalNotifikasi)
				r.Get("/notifikasi/pengaturan", h.GetPengaturanNotifikasi)
				r.Post("/notifikasi/pengaturan", h.SetPengaturanNotifikasi)
				r.Post("/pengingat-absensi", h.SetPengingatAbsensi)
				r.Post("/pengingat-absensi/kirim", h.KirimPengingatAbsensiSekarang)

				// Pengingat SPP bulanan (daftar santri yang belum bayar)
				r.Get("/pengingat-spp", h.GetPengingatSPP)
				r.Post("/pengingat-spp", h.SetPengingatSPP)
				r.Post("/pengingat-spp/kirim", h.KirimPengingatSPP)

				// Telegram (kanal notifikasi resmi)
				r.Get("/telegram/pengaturan", h.GetPengaturanTelegram)
				r.Post("/telegram/pengaturan", h.SetPengaturanTelegram)
				r.Post("/telegram/uji", h.UjiKirimTelegram)

				// Absensi guru + rekap (bulan/semester/tahun)
				r.Get("/absensi-guru", h.GetAbsensiGuru)
				r.Post("/absensi-guru/batch", h.SaveAbsensiGuru)
				r.Get("/absensi-guru/rekap", h.RekapAbsensiGuru)
				r.Get("/absensi-guru/export", h.ExportAbsensiGuru)

				// ===== KHUSUS SUPERADMIN =====
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("superadmin"))
					r.Post("/users/{id}/reset-password", h.ResetPasswordUser)
					r.Delete("/sesi/{id}", h.PutusSesi)
				})
			})
		})
	})

	addr := cfg.AppHost + ":" + cfg.AppPort
	log.Printf("SIM-Madrasah backend berjalan di %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
