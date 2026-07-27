-- 012_whatsapp.sql — pemetaan akun guru ke nomor WhatsApp (untuk bot-login)
-- Kolom telegram_user_id sengaja dibiarkan; dihapus di migrasi terpisah
-- setelah bot WhatsApp terbukti jalan (lihat spec W6).
USE sim_madrasah;
ALTER TABLE users ADD COLUMN whatsapp_number VARCHAR(20) NULL UNIQUE AFTER role;
