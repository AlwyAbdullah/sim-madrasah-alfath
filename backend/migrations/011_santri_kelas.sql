-- SIM-Madrasah — riwayat kelas santri per periode (agar naik kelas / tahun ajaran baru
-- tidak mengubah rapor & leger periode lama).
-- Jalankan: mysql -u root -p sim_madrasah < migrations/011_santri_kelas.sql
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS santri_kelas (
  santri_id   BIGINT NOT NULL,
  periode_id  BIGINT NOT NULL,
  kelas_id    BIGINT NOT NULL,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (santri_id, periode_id),
  CONSTRAINT fk_sk_santri  FOREIGN KEY (santri_id)  REFERENCES santri(id)  ON DELETE CASCADE,
  CONSTRAINT fk_sk_periode FOREIGN KEY (periode_id) REFERENCES periode(id) ON DELETE CASCADE,
  CONSTRAINT fk_sk_kelas   FOREIGN KEY (kelas_id)   REFERENCES kelas(id)   ON DELETE CASCADE,
  INDEX idx_sk_kelas_periode (kelas_id, periode_id)
) ENGINE=InnoDB;

-- Backfill dari nilai yang sudah ada. Karena belum pernah ada kenaikan kelas,
-- kelas santri SEKARANG = kelas historis untuk seluruh periode yang sudah dinilai.
INSERT IGNORE INTO santri_kelas (santri_id, periode_id, kelas_id)
  SELECT DISTINCT n.santri_id, n.periode_id, s.kelas_id
  FROM nilai n JOIN santri s ON s.id = n.santri_id;
