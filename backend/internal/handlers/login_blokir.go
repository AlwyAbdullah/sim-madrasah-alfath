package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/audit"
	"sim-madrasah/backend/internal/httpx"
)

// lamaTerbaca mengubah sisa waktu jadi kalimat pendek berbahasa Indonesia.
// Guru tidak perlu membaca "13m42.318s".
func lamaTerbaca(d time.Duration) string {
	if d < time.Minute {
		return "kurang dari 1 menit"
	}
	return fmt.Sprintf("%d menit", int(d.Minutes())+1)
}

type blokirOut struct {
	Jenis     string `json:"jenis"` // "akun" | "ip"
	Kunci     string `json:"kunci"`
	Nama      string `json:"nama"` // nama pemilik akun, bila kuncinya nama akun
	Gagal     int    `json:"gagal"`
	SisaDetik int    `json:"sisa_detik"`
	SisaTeks  string `json:"sisa_teks"`
}

// GET /login-blokir — siapa saja yang sedang tidak bisa login karena salah
// password berkali-kali.
func (h *Handler) ListBlokirLogin(w http.ResponseWriter, r *http.Request) {
	out := []blokirOut{}
	for _, t := range h.Gembok.Daftar() {
		b := blokirOut{
			Jenis: t.Jenis, Kunci: t.Kunci, Gagal: t.Gagal, SisaDetik: t.SisaDetik,
			SisaTeks: lamaTerbaca(time.Duration(t.SisaDetik) * time.Second),
		}
		if t.Jenis == "akun" {
			var nama string
			if err := h.DB.QueryRow(`SELECT nama FROM users WHERE username = ?`, t.Kunci).Scan(&nama); err == nil {
				b.Nama = strings.TrimSpace(nama)
			}
		}
		out = append(out, b)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type bukaBlokirReq struct {
	Jenis string `json:"jenis"` // "akun" (bawaan) | "ip"
	Kunci string `json:"kunci"`
}

// POST /login-blokir/buka — admin membuka kunci login.
//
// Ini yang dipakai saat seorang guru menelepon "saya tidak bisa masuk":
// membuka kuncinya seketika, tanpa harus menunggu jendela waktunya habis dan
// tanpa perlu mereset password (yang justru menambah satu hal baru untuk diingat).
func (h *Handler) BukaBlokirLogin(w http.ResponseWriter, r *http.Request) {
	var req bukaBlokirReq
	if !decode(w, r, &req) {
		return
	}
	kunci := strings.TrimSpace(req.Kunci)
	if kunci == "" {
		httpx.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Kunci wajib diisi")
		return
	}

	var dibuka bool
	var ringkasan string
	if req.Jenis == "ip" {
		dibuka = h.Gembok.BukaIP(kunci)
		ringkasan = fmt.Sprintf("Membuka kunci login untuk alamat %s", kunci)
	} else {
		dibuka = h.Gembok.BukaAkun(kunci)
		ringkasan = fmt.Sprintf("Membuka kunci login akun %s", kunci)
	}

	if !dibuka {
		// Bukan error: kuncinya bisa saja sudah lepas sendiri sedetik sebelumnya.
		httpx.JSON(w, http.StatusOK, map[string]string{
			"message": fmt.Sprintf("%s memang sedang tidak terkunci — sekarang sudah bisa login.", kunci),
		})
		return
	}
	audit.Catat(h.DB, r, audit.Entri{
		Aksi: audit.BukaBlokir, Entitas: "login", EntitasID: kunci, Ringkasan: ringkasan,
	})
	httpx.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Kunci login %s dibuka. Silakan coba masuk lagi.", kunci),
	})
}
