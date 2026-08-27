package notifworker

import (
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

// RunTelegramMasuk membaca pesan MASUK ke bot, semata-mata untuk menautkan akun
// SIM ke chat Telegram penggunanya.
//
// Alurnya: pengguna menekan tombol di halaman "Hubungkan Telegram", mendapat
// kode sekali pakai, lalu mengirimnya ke bot. Pesan itulah yang membuktikan
// chat tersebut memang miliknya — tanpa langkah ini admin harus menyalin id
// numerik Telegram satu per satu, yang tidak realistis untuk 20 guru.
//
// Bot memakai privacy mode bawaan, jadi di grup ia hanya melihat pesan yang
// diawali "/" atau yang menyebut namanya — obrolan guru tidak terbaca.
func RunTelegramMasuk(db *sql.DB, cfg *config.Config) {
	if cfg.TelegramBotToken == "" {
		return // pengirim Telegram tidak aktif; tidak ada gunanya membaca pesan masuk
	}
	interval := time.Duration(cfg.NotifPollSeconds) * time.Second
	if interval <= 0 || interval > 30*time.Second {
		// penautan terasa "hidup" kalau balasannya cepat; 1 menit terlalu lama
		interval = 10 * time.Second
	}
	client := &http.Client{Timeout: 25 * time.Second}

	simpanBotUsername(db, cfg, client)
	log.Printf("telegram-masuk: aktif — memeriksa pesan tiap %s", interval)

	for {
		if err := prosesPesanMasuk(db, cfg, client); err != nil {
			log.Printf("telegram-masuk: %v", err)
		}
		time.Sleep(interval)
	}
}

// simpanBotUsername menyimpan username bot agar frontend bisa menyusun tautan
// t.me tanpa perlu tahu tokennya.
func simpanBotUsername(db *sql.DB, cfg *config.Config, client *http.Client) {
	body, err := panggilTelegram(client, cfg, "getMe", nil)
	if err != nil {
		log.Printf("telegram-masuk: gagal getMe: %v", err)
		return
	}
	var res struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &res); err != nil || !res.OK || res.Result.Username == "" {
		return
	}
	if _, err := db.Exec(`
		INSERT INTO telegram_pengaturan (id, bot_username) VALUES (1, ?)
		ON DUPLICATE KEY UPDATE bot_username = VALUES(bot_username)`, res.Result.Username); err != nil {
		log.Printf("telegram-masuk: gagal menyimpan bot_username: %v", err)
	}
}

type pesanMasuk struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID        int64  `json:"id"`
			Type      string `json:"type"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
		} `json:"chat"`
	} `json:"message"`
}

func prosesPesanMasuk(db *sql.DB, cfg *config.Config, client *http.Client) error {
	var offset int64
	_ = db.QueryRow(`SELECT update_offset FROM telegram_pengaturan WHERE id = 1`).Scan(&offset)

	body, err := panggilTelegram(client, cfg, "getUpdates", map[string]interface{}{
		"offset":          offset,
		"timeout":         0,
		"allowed_updates": []string{"message"},
	})
	if err != nil {
		return err
	}
	var res struct {
		OK     bool         `json:"ok"`
		Result []pesanMasuk `json:"result"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("balasan getUpdates tidak terbaca: %w", err)
	}
	if !res.OK || len(res.Result) == 0 {
		return nil
	}

	maksID := offset
	for _, u := range res.Result {
		if u.UpdateID >= maksID {
			maksID = u.UpdateID + 1
		}
		if u.Message == nil || u.Message.Chat.Type != "private" {
			continue // penautan hanya lewat chat pribadi
		}
		kode := ambilKode(u.Message.Text)
		if kode == "" {
			continue
		}
		nama := strings.TrimSpace(u.Message.Chat.FirstName + " " + u.Message.Chat.LastName)
		if u.Message.Chat.Username != "" {
			nama = strings.TrimSpace(nama + " (@" + u.Message.Chat.Username + ")")
		}
		balas := tautkanAkun(db, kode, u.Message.Chat.ID, nama)
		if err := kirimTelegram(client, cfg, fmt.Sprintf("%d", u.Message.Chat.ID), balas); err != nil {
			log.Printf("telegram-masuk: gagal membalas %d: %v", u.Message.Chat.ID, err)
		}
	}

	// Offset disimpan SETELAH seluruh pesan diproses. Kalau disimpan lebih dulu
	// lalu terjadi galat, pesannya hilang tanpa pernah ditangani.
	if maksID != offset {
		if _, err := db.Exec(`
			INSERT INTO telegram_pengaturan (id, update_offset) VALUES (1, ?)
			ON DUPLICATE KEY UPDATE update_offset = VALUES(update_offset)`, maksID); err != nil {
			return fmt.Errorf("gagal menyimpan offset: %w", err)
		}
	}
	return nil
}

// ambilKode menerima "/start ABC123", "/mulai ABC123", atau kodenya saja.
func ambilKode(teks string) string {
	t := strings.TrimSpace(teks)
	if t == "" {
		return ""
	}
	bagian := strings.Fields(t)
	kandidat := bagian[len(bagian)-1]
	if strings.HasPrefix(kandidat, "/") {
		return "" // hanya perintah tanpa kode, mis. "/start" polos
	}
	kandidat = strings.ToUpper(strings.TrimSpace(kandidat))
	if len(kandidat) < 4 || len(kandidat) > 16 {
		return ""
	}
	return kandidat
}

// tautkanAkun menukar kode dengan penautan, lalu mengembalikan teks balasan
// untuk pengirimnya. Selalu mengembalikan kalimat yang bisa dibaca orang awam —
// yang menerimanya adalah guru, bukan pengembang.
func tautkanAkun(db *sql.DB, kode string, chatID int64, namaTelegram string) string {
	var userID int64
	var nama string
	err := db.QueryRow(`
		SELECT id, nama FROM users
		WHERE telegram_kode = ? AND telegram_kode_exp > NOW() AND is_active = 1`, kode).Scan(&userID, &nama)
	if err == sql.ErrNoRows {
		return "Kode tidak dikenali atau sudah kedaluwarsa.\n\n" +
			"Buka halaman *Hubungkan Telegram* di SIM Madrasah untuk mengambil kode baru."
	}
	if err != nil {
		log.Printf("telegram-masuk: gagal mencari kode: %v", err)
		return "Maaf, sedang ada gangguan. Coba lagi beberapa saat lagi."
	}

	// Satu chat Telegram hanya boleh terhubung ke satu akun; kalau tidak,
	// pengingat bisa terkirim ke orang yang salah.
	if _, err := db.Exec(`
		UPDATE users SET telegram_user_id = NULL, telegram_nama = NULL
		WHERE telegram_user_id = ? AND id <> ?`, chatID, userID); err != nil {
		log.Printf("telegram-masuk: gagal melepas tautan lama: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE users
		   SET telegram_user_id = ?, telegram_nama = ?, telegram_kode = NULL, telegram_kode_exp = NULL
		 WHERE id = ?`, chatID, nullJikaKosong(namaTelegram), userID); err != nil {
		log.Printf("telegram-masuk: gagal menautkan akun %d: %v", userID, err)
		return "Maaf, gagal menyimpan penautan. Coba lagi."
	}

	log.Printf("telegram-masuk: akun %d (%s) tertaut ke chat %d", userID, strings.TrimSpace(nama), chatID)
	return fmt.Sprintf(
		"Assalamu'alaikum, *%s*.\n\nAkun SIM Madrasah Anda sudah terhubung ke Telegram ini.\n\n"+
			"Mulai sekarang pengingat absensi kelas yang Anda ampu akan dikirim ke sini, "+
			"bukan lagi ke grup.", strings.TrimSpace(nama))
}

func nullJikaKosong(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// panggilTelegram memanggil satu metode Bot API dan mengembalikan bodinya.
func panggilTelegram(client *http.Client, cfg *config.Config, metode string, payload map[string]interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s/bot%s/%s", strings.TrimRight(cfg.TelegramAPIURL, "/"), cfg.TelegramBotToken, metode)

	var resp *http.Response
	var err error
	if payload == nil {
		resp, err = client.Get(url)
	} else {
		b, e := json.Marshal(payload)
		if e != nil {
			return nil, e
		}
		resp, err = client.Post(url, "application/json", strings.NewReader(string(b)))
	}
	if err != nil {
		return nil, fmt.Errorf("tidak bisa menghubungi Telegram: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 200*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Telegram balas %d: %s", resp.StatusCode, ringkas(string(body)))
	}
	return body, nil
}
