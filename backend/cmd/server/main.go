package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"sim-madrasah/backend/internal/config"
	"sim-madrasah/backend/internal/db"
	"sim-madrasah/backend/internal/handlers"
	"sim-madrasah/backend/internal/middleware"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("gagal koneksi DB: %v", err)
	}
	defer conn.Close()

	h := handlers.New(conn, cfg)

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
		// Publik + rate limit untuk login
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(5, 15*time.Minute))
			r.Post("/auth/login", h.Login)
		})
		r.Post("/auth/logout", h.Logout)
		r.Post("/auth/bot-login", h.BotLogin)

		// Terproteksi (semua guru bisa akses semua kelas — tanpa batasan kelas ampu)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWTSecret))

			r.Get("/auth/me", h.Me)

			// Master (baca — semua role login)
			r.Get("/kelas", h.ListKelas)
			r.Get("/kelas/{id}/mapel", h.ListKelasMapel)
			r.Get("/mata-pelajaran", h.ListMapel)
			r.Get("/periode", h.ListPeriode)
			r.Get("/santri", h.ListSantri)

			// Dashboard
			r.Get("/dashboard/summary", h.Summary)
			r.Get("/santri/{id}/detail", h.SantriDetail)

			// Rapor
			r.Get("/rapor", h.RaporData)

			// Absensi
			r.Get("/absensi", h.GetAbsensi)
			r.Post("/absensi/batch", h.SaveAbsensi)
			r.Get("/absensi/export", h.ExportAbsensi)

			// Nilai
			r.Get("/nilai", h.GetNilai)
			r.Post("/nilai/batch", h.SaveNilai)
			r.Get("/nilai/export", h.ExportNilai)
			r.Get("/nilai/leger", h.LegerNilai)
			r.Get("/nilai/leger/export", h.ExportLeger)

			// ===== KHUSUS ADMIN =====
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))

				// SPP (tidak boleh dilihat guru)
				r.Get("/spp", h.GetSPP)
				r.Post("/spp/toggle", h.ToggleSPP)
				r.Get("/spp/export", h.ExportSPP)

				// Master CRUD
				r.Post("/kelas", h.CreateKelas)
				r.Put("/kelas/{id}", h.UpdateKelas)
				r.Delete("/kelas/{id}", h.DeleteKelas)
				r.Put("/kelas/{id}/mapel", h.SetKelasMapel)

				r.Post("/santri", h.CreateSantri)
				r.Put("/santri/{id}", h.UpdateSantri)
				r.Delete("/santri/{id}", h.DeleteSantri)
				r.Post("/santri/import", h.ImportSantri)

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
			})
		})
	})

	addr := ":" + cfg.AppPort
	log.Printf("SIM-Madrasah backend berjalan di %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
