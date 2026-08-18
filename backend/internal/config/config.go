package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort         string
	AppEnv          string
	CorsOrigin      string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	JWTSecret       string
	JWTExpiryMin    int
	CookieSecure    bool
	BotSharedSecret string

	// Worker pengirim notifikasi WhatsApp (opsional). Kosong = worker tidak aktif;
	// endpoint /notifikasi/pending & /notifikasi/status tetap bisa dipakai otomasi
	// eksternal (mis. n8n) sebagai gantinya.
	WahaURL              string
	WahaSession          string
	WahaAPIKey           string
	WahaPollSeconds      int
	WahaSendDelaySeconds int
	WahaBatchLimit       int

	// Telegram Bot API (resmi & gratis). Kosong = pengirim Telegram tidak aktif.
	// Token didapat dari @BotFather; tujuan chat diatur dari halaman admin.
	TelegramBotToken string
	TelegramAPIURL   string // dapat diarahkan ke server tiruan saat pengujian
}

func Load() *Config {
	// .env opsional — abaikan error bila tidak ada (pakai env sistem)
	_ = godotenv.Load()

	expiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_MINUTES", "60"))
	secure, _ := strconv.ParseBool(getEnv("COOKIE_SECURE", "false"))

	return &Config{
		AppPort:         getEnv("APP_PORT", "8080"),
		AppEnv:          getEnv("APP_ENV", "development"),
		CorsOrigin:      getEnv("CORS_ORIGIN", "http://localhost:3000"),
		DBHost:          getEnv("DB_HOST", "127.0.0.1"),
		DBPort:          getEnv("DB_PORT", "3306"),
		DBUser:          getEnv("DB_USER", "root"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", "sim_madrasah"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiryMin:    expiry,
		CookieSecure:    secure,
		BotSharedSecret: getEnv("BOT_SHARED_SECRET", ""),

		WahaURL:              getEnv("WAHA_URL", ""),
		WahaSession:          getEnv("WAHA_SESSION", "default"),
		WahaAPIKey:           getEnv("WAHA_API_KEY", ""),
		WahaPollSeconds:      getEnvInt("WAHA_POLL_SECONDS", 60),
		WahaSendDelaySeconds: getEnvInt("WAHA_SEND_DELAY_SECONDS", 8),
		WahaBatchLimit:       getEnvInt("WAHA_BATCH_LIMIT", 20),

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
