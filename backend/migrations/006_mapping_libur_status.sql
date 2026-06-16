-- SIM-Madrasah — pemetaan pelajaran per kelas, hari libur, status kelas, no ortu
USE sim_madrasah;

-- Nomor orang tua (opsional)
ALTER TABLE santri ADD COLUMN no_ortu VARCHAR(30) NULL AFTER jenis_kelamin;

-- Status aktif kelas (Alumni / Waqof = 0 → disembunyikan dari daftar aktif)
ALTER TABLE kelas ADD COLUMN aktif TINYINT(1) NOT NULL DEFAULT 1;

-- Pemetaan pelajaran per kelas + kitab per (kelas, pelajaran)
CREATE TABLE IF NOT EXISTS kelas_mapel (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  kelas_id          BIGINT NOT NULL,
  mata_pelajaran_id BIGINT NOT NULL,
  kitab             VARCHAR(120) NULL,
  urutan            INT NOT NULL DEFAULT 0,
  UNIQUE KEY uq_kelas_mapel (kelas_id, mata_pelajaran_id),
  CONSTRAINT fk_km_kelas FOREIGN KEY (kelas_id) REFERENCES kelas(id) ON DELETE CASCADE,
  CONSTRAINT fk_km_mapel FOREIGN KEY (mata_pelajaran_id) REFERENCES mata_pelajaran(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Hari libur (dikelola admin) — dikecualikan dari perhitungan kehadiran
CREATE TABLE IF NOT EXISTS hari_libur (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  tanggal    DATE NOT NULL UNIQUE,
  keterangan VARCHAR(255) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Kelas khusus
INSERT INTO kelas (nama, tingkat, aktif) VALUES
  ('Alumni', '-', 0), ('Waqof / Berhenti', '-', 0)
ON DUPLICATE KEY UPDATE aktif = VALUES(aktif);

-- Periode aktif → Masehi
UPDATE periode SET nama = '2025/2026 Ganjil', tahun_ajaran = '2025/2026' WHERE is_active = 1;
