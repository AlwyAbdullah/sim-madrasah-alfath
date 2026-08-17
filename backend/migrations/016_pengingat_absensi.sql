-- 016_pengingat_absensi.sql — pengingat harian bila masih ada kelas yang belum diabsen.
--
-- Jamnya sengaja disimpan di database (bukan dipaku di kode) supaya admin bisa
-- mengubahnya sendiri dari halaman tanpa perlu deploy ulang — jadwal madrasah
-- berubah saat Ramadan/pekan ujian.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS pengingat_absensi_pengaturan (
  id             TINYINT PRIMARY KEY DEFAULT 1,
  aktif          TINYINT(1) NOT NULL DEFAULT 1,
  jam            TINYINT    NOT NULL DEFAULT 19,   -- 0..23
  menit          TINYINT    NOT NULL DEFAULT 0,    -- 0..59
  -- tanggal terakhir pengingat dikirim; dipakai agar tidak terkirim dua kali sehari
  terakhir_kirim DATE       NULL,
  updated_at     DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

INSERT IGNORE INTO pengingat_absensi_pengaturan (id, aktif, jam, menit) VALUES (1, 1, 19, 0);
