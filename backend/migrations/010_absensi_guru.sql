-- SIM-Madrasah — absensi guru (rekap bulanan/semester/tahunan).
-- Jalankan: mysql -u root -p sim_madrasah < migrations/010_absensi_guru.sql
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS absensi_guru (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  guru_id     BIGINT NOT NULL,
  tanggal     DATE   NOT NULL,
  status      ENUM('hadir','izin','sakit','alpha') NOT NULL,
  keterangan  VARCHAR(255) NULL,
  created_by  BIGINT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_absensi_guru (guru_id, tanggal),
  CONSTRAINT fk_ag_guru FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE CASCADE,
  CONSTRAINT fk_ag_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_ag_tanggal (tanggal)
) ENGINE=InnoDB;
