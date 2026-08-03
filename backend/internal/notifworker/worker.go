// Package notifworker mengirim antrean notifikasi WhatsApp (tabel notifikasi_wa)
// langsung ke WAHA — pengganti bawaan untuk otomasi eksternal (n8n/skrip cron).
//
// Tidak aktif bila WAHA_URL kosong di .env; endpoint /notifikasi/pending dan
// /notifikasi/status tetap berfungsi seperti biasa untuk siapa pun yang lebih
// suka memakai n8n atau skrip sendiri.
package notifworker

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"sim-madrasah/backend/internal/config"
)

type pendingItem struct {
	ID     int64
	Tujuan string
	Pesan  string
}

// Run memblokir selamanya (dipanggil lewat `go notifworker.Run(...)`).
// Sengaja pakai loop sleep, bukan ticker: memproses satu batch penuh dulu
// (termasuk jeda antar pesan) sebelum menunggu giliran berikutnya, sehingga
// tidak pernah ada dua batch berjalan bersamaan.
func Run(db *sql.DB, cfg *config.Config) {
	if cfg.WahaURL == "" {
		log.Println("notifworker: WAHA_URL kosong -> worker pengirim WA tidak aktif (pakai n8n/skrip lain bila perlu)")
		return
	}
	interval := time.Duration(cfg.WahaPollSeconds) * time.Second
	log.Printf("notifworker: aktif — polling tiap %s, kirim ke %s (sesi %q)", interval, cfg.WahaURL, cfg.WahaSession)

	client := &http.Client{Timeout: 20 * time.Second}
	for {
		prosesBatch(db, cfg, client)
		time.Sleep(interval)
	}
}

func prosesBatch(db *sql.DB, cfg *config.Config, client *http.Client) {
	items, err := ambilPending(db, cfg.WahaBatchLimit)
	if err != nil {
		log.Printf("notifworker: gagal ambil antrean: %v", err)
		return
	}
	for i, it := range items {
		if err := kirimWaha(client, cfg, it); err != nil {
			tandaiGagal(db, it.ID, err)
			log.Printf("notifworker: GAGAL kirim id=%d ke %s: %v", it.ID, it.Tujuan, err)
		} else {
			tandaiTerkirim(db, it.ID)
			log.Printf("notifworker: terkirim id=%d ke %s", it.ID, it.Tujuan)
		}
		if i < len(items)-1 {
			time.Sleep(time.Duration(cfg.WahaSendDelaySeconds) * time.Second)
		}
	}
}

func ambilPending(db *sql.DB, limit int) ([]pendingItem, error) {
	rows, err := db.Query(`SELECT id, tujuan, pesan FROM notifikasi_wa WHERE status = 'pending' ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pendingItem
	for rows.Next() {
		var it pendingItem
		if err := rows.Scan(&it.ID, &it.Tujuan, &it.Pesan); err == nil {
			out = append(out, it)
		}
	}
	return out, nil
}

func tandaiTerkirim(db *sql.DB, id int64) {
	if _, err := db.Exec(`
		UPDATE notifikasi_wa SET status='terkirim', dikirim_at=NOW(), percobaan=percobaan+1, catatan=NULL
		WHERE id=?`, id); err != nil {
		log.Printf("notifworker: gagal menandai terkirim id=%d: %v", id, err)
	}
}

func tandaiGagal(db *sql.DB, id int64, sebab error) {
	catatan := sebab.Error()
	if len(catatan) > 255 {
		catatan = catatan[:255]
	}
	if _, err := db.Exec(`
		UPDATE notifikasi_wa SET status='gagal', percobaan=percobaan+1, catatan=?
		WHERE id=?`, catatan, id); err != nil {
		log.Printf("notifworker: gagal menandai gagal id=%d: %v", id, err)
	}
}

func kirimWaha(client *http.Client, cfg *config.Config, it pendingItem) error {
	body, err := json.Marshal(map[string]string{
		"session": cfg.WahaSession,
		"chatId":  it.Tujuan + "@c.us",
		"text":    it.Pesan,
	})
	if err != nil {
		return err
	}
	url := strings.TrimRight(cfg.WahaURL, "/") + "/api/sendText"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.WahaAPIKey != "" {
		req.Header.Set("X-Api-Key", cfg.WahaAPIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tidak bisa menghubungi WAHA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("WAHA balas %d: %s", resp.StatusCode, strings.TrimSpace(string(limited)))
	}
	return nil
}
