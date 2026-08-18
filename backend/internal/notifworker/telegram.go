// Package notifworker mengirim antrean notifikasi (tabel `notifikasi`) lewat
// Telegram Bot API, dan menyusun pengingat berkala (absensi harian, SPP bulanan).
//
// Telegram adalah SATU-SATUNYA kanal pengiriman. WhatsApp pernah dipakai lewat
// WAHA (API tidak resmi) dan dihapus seluruhnya: nomornya berisiko dibatasi,
// dan memang sempat terjadi.
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

// RunTelegram memblokir selamanya (dipanggil lewat `go notifworker.RunTelegram(...)`).
// Sengaja pakai loop sleep, bukan ticker: satu batch penuh diproses dulu (termasuk
// jeda antar pesan) sebelum menunggu giliran berikutnya, sehingga tidak pernah ada
// dua batch berjalan bersamaan.
//
// Saklar aktif/nonaktif (tabel notifikasi_pengaturan, diatur dari halaman admin)
// dicek tiap putaran — bisa dimatikan/dinyalakan tanpa restart backend. Saat
// nonaktif, pesan tetap diantrekan seperti biasa; worker cuma diam.
func RunTelegram(db *sql.DB, cfg *config.Config) {
	if cfg.TelegramBotToken == "" {
		log.Println("telegram: TELEGRAM_BOT_TOKEN kosong -> pengirim Telegram tidak aktif")
		return
	}
	interval := time.Duration(cfg.NotifPollSeconds) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	log.Printf("telegram: siap — polling tiap %s", interval)

	client := &http.Client{Timeout: 20 * time.Second}
	var lastAktif *bool
	for {
		aktif := pengirimanAktif(db)
		if lastAktif == nil || *lastAktif != aktif {
			if aktif {
				log.Println("telegram: pengiriman DIAKTIFKAN (pengaturan admin)")
			} else {
				log.Println("telegram: pengiriman DIJEDA (pengaturan admin) — pesan tetap mengantre")
			}
			lastAktif = &aktif
		}
		if aktif {
			prosesBatchTelegram(db, cfg, client)
		}
		time.Sleep(interval)
	}
}

// pengirimanAktif membaca saklar dari database. Bila baris/tabel belum ada
// (mis. migrasi belum jalan), dianggap aktif — supaya deploy yang tertinggal
// tidak diam-diam berhenti mengirim.
func pengirimanAktif(db *sql.DB) bool {
	var aktif bool
	if err := db.QueryRow(`SELECT aktif FROM notifikasi_pengaturan WHERE id = 1`).Scan(&aktif); err != nil {
		return true
	}
	return aktif
}

func prosesBatchTelegram(db *sql.DB, cfg *config.Config, client *http.Client) {
	rows, err := db.Query(`
		SELECT id, tujuan, pesan FROM notifikasi
		WHERE status = 'pending'
		ORDER BY id LIMIT ?`, cfg.NotifBatchLimit)
	if err != nil {
		log.Printf("telegram: gagal ambil antrean: %v", err)
		return
	}
	var items []pendingItem
	for rows.Next() {
		var it pendingItem
		if err := rows.Scan(&it.ID, &it.Tujuan, &it.Pesan); err == nil {
			items = append(items, it)
		}
	}
	rows.Close()

	for i, it := range items {
		if err := kirimTelegram(client, cfg, it.Tujuan, it.Pesan); err != nil {
			tandaiGagal(db, it.ID, err)
			log.Printf("telegram: GAGAL kirim id=%d ke %s: %v", it.ID, it.Tujuan, err)
		} else {
			tandaiTerkirim(db, it.ID)
			log.Printf("telegram: terkirim id=%d ke %s", it.ID, it.Tujuan)
		}
		if i < len(items)-1 {
			time.Sleep(time.Duration(cfg.NotifSendDelaySeconds) * time.Second)
		}
	}
}

func tandaiTerkirim(db *sql.DB, id int64) {
	if _, err := db.Exec(`
		UPDATE notifikasi SET status='terkirim', dikirim_at=NOW(), percobaan=percobaan+1, catatan=NULL
		WHERE id=?`, id); err != nil {
		log.Printf("telegram: gagal menandai terkirim id=%d: %v", id, err)
	}
}

func tandaiGagal(db *sql.DB, id int64, sebab error) {
	catatan := sebab.Error()
	if len(catatan) > 255 {
		catatan = catatan[:255]
	}
	if _, err := db.Exec(`
		UPDATE notifikasi SET status='gagal', percobaan=percobaan+1, catatan=?
		WHERE id=?`, catatan, id); err != nil {
		log.Printf("telegram: gagal menandai gagal id=%d: %v", id, err)
	}
}

// kirimTelegram mengirim satu pesan. Pesan kita memakai penanda *tebal* / _miring_
// yang kebetulan sama dengan Markdown Telegram. Bila Telegram menolak karena ada
// karakter yang mengacaukan Markdown (mis. nama tabel bergaris bawah), dicoba
// ulang sebagai teks biasa supaya pesan tetap sampai — isi lebih penting daripada
// gaya huruf.
func kirimTelegram(client *http.Client, cfg *config.Config, chatID, pesan string) error {
	if chatID == "" {
		return fmt.Errorf("chat_id Telegram belum diatur")
	}
	err := postTelegram(client, cfg, chatID, pesan, "Markdown")
	if err != nil && galatPenguraian(err) {
		log.Printf("telegram: Markdown ditolak, mengirim ulang sebagai teks biasa")
		return postTelegram(client, cfg, chatID, pesan, "")
	}
	return err
}

// galatPenguraian membedakan "Markdown-nya bermasalah" dari kegagalan lain
// (mis. chat not found / bot dikeluarkan dari grup). Hanya galat penguraian yang
// layak dicoba ulang sebagai teks biasa — selain itu mengulang hanya membuang waktu.
func galatPenguraian(err error) bool {
	p := strings.ToLower(err.Error())
	if !strings.Contains(p, "400") {
		return false
	}
	return strings.Contains(p, "parse") || strings.Contains(p, "entit")
}

func postTelegram(client *http.Client, cfg *config.Config, chatID, pesan, parseMode string) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    pesan,
		// jangan menampilkan pratinjau tautan agar pesan tetap ringkas
		"disable_web_page_preview": true,
	}
	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(cfg.TelegramAPIURL, "/"), cfg.TelegramBotToken)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tidak bisa menghubungi Telegram: %w", err)
	}
	defer resp.Body.Close()

	isi, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// tampilkan pesan dari Telegram apa adanya agar mudah ditindaklanjuti
		// (mis. "chat not found" = bot belum dimasukkan ke grup / belum /start)
		return fmt.Errorf("Telegram balas %d: %s", resp.StatusCode, ringkas(string(isi)))
	}
	return nil
}

func ringkas(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
