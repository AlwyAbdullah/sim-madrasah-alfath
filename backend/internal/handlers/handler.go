package handlers

import (
	"database/sql"

	"sim-madrasah/backend/internal/config"
	"sim-madrasah/backend/internal/gembok"
)

type Handler struct {
	DB  *sql.DB
	Cfg *config.Config
	// Gembok menahan percobaan login yang gagal berulang. Disimpan di Handler
	// (bukan sebagai middleware terpisah) karena penguncian kini per NAMA AKUN,
	// dan nama akun baru diketahui setelah badan permintaan dibaca.
	Gembok *gembok.Gembok
}

func New(db *sql.DB, cfg *config.Config) *Handler {
	return &Handler{DB: db, Cfg: cfg, Gembok: gembok.Bawaan()}
}
