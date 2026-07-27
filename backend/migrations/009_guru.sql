-- SIM-Madrasah — master data guru (untuk honor/identitas guru).
-- Jalankan: mysql -u root -p sim_madrasah < migrations/009_guru.sql
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS guru (
  id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
  nama               VARCHAR(120) NOT NULL,
  no_rekening        VARCHAR(50)  NULL,
  nama_bank          VARCHAR(80)  NULL,
  mengajar_per_pekan INT          NULL,           -- jumlah mengajar dalam 1 pekan
  no_telepon         VARCHAR(30)  NULL,
  created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_guru_nama (nama)
) ENGINE=InnoDB;
