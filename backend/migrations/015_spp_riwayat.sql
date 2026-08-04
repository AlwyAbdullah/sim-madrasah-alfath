-- 015_spp_riwayat.sql — jejak audit + fondasi "Kembalikan" (undo setelah simpan) untuk SPP.
--
-- Setiap aksi simpan dari halaman SPP menghasilkan satu `batch_id` (UUID) dan
-- satu baris riwayat per sel yang berubah, lengkap dengan nilai LAMA dan BARU.
-- Dengan begitu satu aksi simpan bisa dikembalikan utuh, dan selalu terlihat
-- siapa mengubah apa dan kapan.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS spp_riwayat (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  batch_id    CHAR(36)     NOT NULL,          -- satu aksi simpan / kembalikan
  santri_id   BIGINT       NOT NULL,
  tahun       SMALLINT     NOT NULL,          -- tahun KALENDER baris spp
  bulan       TINYINT      NOT NULL,
  lunas_lama  TINYINT(1)   NULL,              -- NULL = baris spp belum pernah ada
  lunas_baru  TINYINT(1)   NOT NULL,
  ket_lama    VARCHAR(255) NULL,
  ket_baru    VARCHAR(255) NULL,
  aksi        ENUM('simpan','kembalikan') NOT NULL DEFAULT 'simpan',
  created_by  BIGINT       NULL,
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_sppr_santri FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  CONSTRAINT fk_sppr_user   FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_sppr_batch (batch_id),
  INDEX idx_sppr_waktu (created_at)
) ENGINE=InnoDB;

-- Ringkasan per aksi simpan: dipakai panel "Riwayat Perubahan" dan penanda
-- batch yang sudah dikembalikan (agar tidak bisa dikembalikan dua kali).
CREATE TABLE IF NOT EXISTS spp_riwayat_batch (
  batch_id     CHAR(36) PRIMARY KEY,
  aksi         ENUM('simpan','kembalikan') NOT NULL DEFAULT 'simpan',
  jumlah_sel   INT      NOT NULL DEFAULT 0,
  tahun_ajaran SMALLINT NOT NULL,             -- tahun ajaran mulai (Juli)
  dikembalikan TINYINT(1) NOT NULL DEFAULT 0,
  asal_batch   CHAR(36) NULL,                 -- diisi bila aksi = 'kembalikan'
  created_by   BIGINT   NULL,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_sppb_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_sppb_waktu (created_at)
) ENGINE=InnoDB;
