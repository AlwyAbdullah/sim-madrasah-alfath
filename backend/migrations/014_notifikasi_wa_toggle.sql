-- 014_notifikasi_wa_toggle.sql — saklar aktif/nonaktif untuk worker pengirim WA.
-- Baris tunggal (id selalu 1). Saat nonaktif, pesan tetap mengantre normal
-- (absensi tidak terpengaruh) tapi worker tidak memanggil WAHA sampai diaktifkan lagi.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS notifikasi_wa_pengaturan (
  id         TINYINT PRIMARY KEY DEFAULT 1,
  aktif      TINYINT(1) NOT NULL DEFAULT 1,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

INSERT IGNORE INTO notifikasi_wa_pengaturan (id, aktif) VALUES (1, 1);
