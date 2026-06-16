-- SIM-Madrasah — modul SPP (pembayaran bulanan)
-- Jalankan: mysql -u root < migrations/004_spp.sql
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS spp (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id     BIGINT   NOT NULL,
  tahun         SMALLINT NOT NULL,           -- mis. 2025
  bulan         TINYINT  NOT NULL,           -- 1..12
  lunas         TINYINT(1) NOT NULL DEFAULT 0,
  nominal       DECIMAL(10,2) NULL,
  tanggal_bayar DATE NULL,
  keterangan    VARCHAR(255) NULL,
  created_by    BIGINT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_spp (santri_id, tahun, bulan),
  CONSTRAINT fk_spp_santri FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  CONSTRAINT fk_spp_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_spp_tahun (tahun)
) ENGINE=InnoDB;
