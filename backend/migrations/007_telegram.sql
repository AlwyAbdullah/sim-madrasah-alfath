-- 007_telegram.sql — pemetaan akun guru ke Telegram (untuk bot-login)
USE sim_madrasah;
ALTER TABLE users ADD COLUMN telegram_user_id BIGINT NULL UNIQUE AFTER role;
