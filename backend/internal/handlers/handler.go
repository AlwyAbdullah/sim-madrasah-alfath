package handlers

import (
	"database/sql"

	"sim-madrasah/backend/internal/config"
)

type Handler struct {
	DB  *sql.DB
	Cfg *config.Config
}

func New(db *sql.DB, cfg *config.Config) *Handler {
	return &Handler{DB: db, Cfg: cfg}
}
