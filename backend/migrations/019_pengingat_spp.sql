-- 019_pengingat_spp.sql — pengingat bulanan daftar santri yang belum bayar SPP.
--
-- Pola sama dengan pengingat absensi (016): jadwalnya disimpan di database, bukan
-- dipaku di kode, supaya admin bisa menggesernya sendiri tanpa deploy ulang.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS pengingat_spp_pengaturan (
  id      TINYINT PRIMARY KEY DEFAULT 1,
  aktif   TINYINT(1) NOT NULL DEFAULT 1,
  -- tanggal berapa tiap bulan pengingat dikirim. Dibatasi 1..28 di sisi aplikasi
  -- supaya tanggalnya selalu ada — tanggal 29-31 tidak ada di Februari.
  tanggal TINYINT NOT NULL DEFAULT 10,
  jam     TINYINT NOT NULL DEFAULT 19,   -- 0..23
  menit   TINYINT NOT NULL DEFAULT 0,    -- 0..59
  -- bulan terakhir pengingat dikirim, format 'YYYY-MM'. Dipakai agar satu bulan
  -- hanya dikirim sekali walau server sempat mati/restart berkali-kali.
  terakhir_kirim CHAR(7) NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

INSERT IGNORE INTO pengingat_spp_pengaturan (id, aktif, tanggal, jam, menit)
VALUES (1, 1, 10, 19, 0);
