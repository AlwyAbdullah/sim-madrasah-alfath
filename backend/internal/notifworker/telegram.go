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

// RunTelegram mengirim antrean notifikasi berkanal 'telegram' lewat Telegram Bot API.
//
// Berbeda dengan WhatsApp (WAHA), API ini RESMI: tidak ada sesi yang bisa tercabut,
// tidak butuh nomor HP, dan tidak ada risiko nomor diblokir. Nonaktif otomatis bila
// TELEGRAM_BOT_TOKEN kosong.
//
// Memakai saklar pengiriman yang sama dengan WhatsApp (notifikasi_wa_pengaturan),
// sehingga satu tombol di halaman admin mengendalikan keduanya.
func RunTelegram(db *sql.DB, cfg *config.Config) {
	if cfg.TelegramBotToken == "" {
		log.Println("telegram: TELEGRAM_BOT_TOKEN kosong -> pengirim Telegram tidak aktif")
		return
	}
	interval := time.Duration(cfg.WahaPollSeconds) * time.Second
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

func prosesBatchTelegram(db *sql.DB, cfg *config.Config, client *http.Client) {
	rows, err := db.Query(`
		SELECT id, tujuan, pesan FROM notifikasi_wa
		WHERE status = 'pending' AND kanal = 'telegram'
		ORDER BY id LIMIT ?`, cfg.WahaBatchLimit)
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
			time.Sleep(time.Duration(cfg.WahaSendDelaySeconds) * time.Second)
		}
	}
}

// kirimTelegram mengirim satu pesan. Pesan kita memakai penanda gaya WhatsApp
// (*tebal*, _miring_) yang kebetulan sama dengan Markdown Telegram. Bila Telegram
// menolak karena ada karakter yang mengacaukan Markdown, dicoba ulang sebagai
// teks biasa supaya pesan tetap sampai (isi lebih penting daripada gaya huruf).
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
// (mis. chat not found / bot diblokir). Hanya galat penguraian yang layak
// dicoba ulang sebagai teks biasa — selain itu mengulang hanya membuang waktu.
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
