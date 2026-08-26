package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// AppHost = alamat yang didengarkan. Bawaannya 127.0.0.1 karena backend ini
	// SELALU diakses lewat nginx (proxy_pass 127.0.0.1:8090); mengikat ke semua
	// antarmuka berarti API-nya bisa dihubungi langsung dari internet — melewati
	// HTTPS dan melewati seluruh penanganan header nginx.
	// Isi "0.0.0.0" hanya bila memang perlu diakses dari mesin lain.
	AppHost      string
	AppPort      string
	AppEnv       string
	CorsOrigin   string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	JWTSecret    string
	JWTExpiryMin int
	CookieSecure bool

	// Irama worker pengirim notifikasi (tabel `notifikasi` → Telegram).
	NotifPollSeconds      int
	NotifSendDelaySeconds int
	NotifBatchLimit       int

	// Telegram Bot API (resmi & gratis) — satu-satunya kanal pengiriman.
	// Kosong = pengirim tidak aktif. Token didapat dari @BotFather;
	// tujuan chat diatur dari halaman admin.
	TelegramBotToken string
	TelegramAPIURL   string // dapat diarahkan ke server tiruan saat pengujian
}

func Load() *Config {
	// .env opsional — abaikan error bila tidak ada (pakai env sistem)
	_ = godotenv.Load()

	expiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_MINUTES", "60"))
	secure, _ := strconv.ParseBool(getEnv("COOKIE_SECURE", "false"))

	return &Config{
		AppHost:      getEnv("APP_HOST", "127.0.0.1"),
		AppPort:      getEnv("APP_PORT", "8080"),
		AppEnv:       getEnv("APP_ENV", "development"),
		CorsOrigin:   getEnv("CORS_ORIGIN", "http://localhost:3000"),
		DBHost:       getEnv("DB_HOST", "127.0.0.1"),
		DBPort:       getEnv("DB_PORT", "3306"),
		DBUser:       getEnv("DB_USER", "root"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		DBName:       getEnv("DB_NAME", "sim_madrasah"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiryMin: expiry,
		CookieSecure: secure,

		NotifPollSeconds:      getEnvInt("NOTIF_POLL_SECONDS", 60),
		NotifSendDelaySeconds: getEnvInt("NOTIF_SEND_DELAY_SECONDS", 3),
		NotifBatchLimit:       getEnvInt("NOTIF_BATCH_LIMIT", 20),

		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramAPIURL:   getEnv("TELEGRAM_API_URL", "https://api.telegram.org"),
	}
}

func (c *Config) DSN() string {
	// allowNativePasswords=true: VPS MySQL pakai mysql_native_password (perlu true).
	// Tetap kompatibel dgn MySQL 8.4 (driver pakai caching_sha2_password bila server memintanya).
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&charset=utf8mb4&allowNativePasswords=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
