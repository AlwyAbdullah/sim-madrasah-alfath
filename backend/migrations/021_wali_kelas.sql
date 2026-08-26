-- 021_wali_kelas.sql — wali kelas, boleh lebih dari satu per kelas.
--
-- `kelas.wali_id` yang lama hanya menampung SATU wali, sedangkan Kelas 1 dan
-- Kelas 2 masing-masing punya dua. Kolom itu juga tidak pernah dipakai (NULL di
-- seluruh 13 kelas), jadi dibuang saja daripada menyisakan dua sumber kebenaran.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS kelas_wali (
  kelas_id   BIGINT  NOT NULL,
  user_id    BIGINT  NOT NULL,
  urutan     TINYINT NOT NULL DEFAULT 1,   -- 1 = wali utama, 2 = pendamping
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (kelas_id, user_id),
  KEY idx_kelas_wali_user (user_id),
  CONSTRAINT fk_kelas_wali_kelas FOREIGN KEY (kelas_id) REFERENCES kelas(id) ON DELETE CASCADE,
  CONSTRAINT fk_kelas_wali_user  FOREIGN KEY (user_id)  REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

ALTER TABLE kelas DROP FOREIGN KEY fk_kelas_wali;
ALTER TABLE kelas DROP COLUMN wali_id;
