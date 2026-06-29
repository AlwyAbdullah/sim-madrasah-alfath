-- 009_catatan_tugas.sql — Fase B: catatan per santri + tugas/PR per kelas
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS catatan (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id BIGINT NOT NULL,
  tanggal DATE NOT NULL,
  teks VARCHAR(500) NOT NULL,
  created_by BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_catatan_santri FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  CONSTRAINT fk_catatan_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_catatan_santri (santri_id), INDEX idx_catatan_tanggal (tanggal)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS tugas (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  kelas_id BIGINT NOT NULL,
  mata_pelajaran_id BIGINT NULL,
  deskripsi VARCHAR(500) NOT NULL,
  tanggal_diberikan DATE NOT NULL,
  tenggat DATE NULL,
  created_by BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_tugas_kelas FOREIGN KEY (kelas_id) REFERENCES kelas(id) ON DELETE CASCADE,
  CONSTRAINT fk_tugas_mapel FOREIGN KEY (mata_pelajaran_id) REFERENCES mata_pelajaran(id),
  CONSTRAINT fk_tugas_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
