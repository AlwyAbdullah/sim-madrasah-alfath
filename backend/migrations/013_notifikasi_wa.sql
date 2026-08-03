-- 013_notifikasi_wa.sql — antrean (outbox) notifikasi WhatsApp ke orang tua.
-- Backend hanya MENGANTRIKAN pesan; bot WhatsApp yang mengambil & mengirim,
-- lalu melapor balik statusnya. Aman bila bot sedang mati: pesan menunggu.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS notifikasi_wa (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id   BIGINT NULL,                     -- konteks (boleh NULL utk pesan umum)
  jenis       VARCHAR(30) NOT NULL,            -- mis. 'absensi_alpha'
  ref_tanggal DATE NULL,                       -- tanggal acuan, utk cegah duplikat
  tujuan      VARCHAR(20) NOT NULL,            -- nomor kanonik 628xxxx
  pesan       VARCHAR(1000) NOT NULL,
  status      ENUM('pending','terkirim','gagal','batal') NOT NULL DEFAULT 'pending',
  percobaan   INT NOT NULL DEFAULT 0,
  catatan     VARCHAR(255) NULL,               -- pesan error dari bot bila gagal
  dikirim_at  DATETIME NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_notif (santri_id, jenis, ref_tanggal),
  CONSTRAINT fk_notif_santri FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  INDEX idx_notif_status (status, id)
) ENGINE=InnoDB;
