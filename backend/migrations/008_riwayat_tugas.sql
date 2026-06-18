-- 008_riwayat_tugas.sql — riwayat Tugas ke-1..n (kolom nilai.tugas = rata-ratanya)
USE sim_madrasah;
CREATE TABLE IF NOT EXISTS riwayat_tugas (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id         BIGINT NOT NULL,
  mata_pelajaran_id BIGINT NOT NULL,
  periode_id        BIGINT NOT NULL,
  ke                INT    NOT NULL,
  nilai             DECIMAL(5,2) NOT NULL,
  created_by        BIGINT NULL,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_riwayat_tugas (santri_id, mata_pelajaran_id, periode_id, ke),
  CONSTRAINT fk_nt_santri  FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  CONSTRAINT fk_nt_mapel   FOREIGN KEY (mata_pelajaran_id) REFERENCES mata_pelajaran(id),
  CONSTRAINT fk_nt_periode FOREIGN KEY (periode_id) REFERENCES periode(id),
  CONSTRAINT fk_nt_user    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
