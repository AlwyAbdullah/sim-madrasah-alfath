-- 017_telegram_kanal.sql — dukungan pengiriman notifikasi lewat Telegram.
--
-- Alasan: WhatsApp lewat WAHA memakai API tidak resmi, sehingga nomor berisiko
-- dibatasi/diblokir (pernah terjadi: 'device_removed' + RESTRICT_ALL_COMPANIONS).
-- Telegram Bot API resmi, gratis, tanpa nomor HP, tanpa sesi yang bisa tercabut.
--
-- Tabel antrean tetap bernama notifikasi_wa agar tidak memecah kode/riwayat yang
-- sudah ada; kolom `kanal` yang menentukan lewat mana pesan dikirim.
USE sim_madrasah;

ALTER TABLE notifikasi_wa
  ADD COLUMN kanal ENUM('whatsapp','telegram') NOT NULL DEFAULT 'whatsapp' AFTER jenis;

-- Tujuan default pengiriman Telegram (grup guru atau chat pribadi admin).
-- Token bot TIDAK disimpan di sini — token ada di .env backend (TELEGRAM_BOT_TOKEN).
CREATE TABLE IF NOT EXISTS telegram_pengaturan (
  id         TINYINT PRIMARY KEY DEFAULT 1,
  chat_id    VARCHAR(64) NULL,   -- mis. -1001234567890 (grup) atau 123456789 (pribadi)
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

INSERT IGNORE INTO telegram_pengaturan (id, chat_id) VALUES (1, NULL);
