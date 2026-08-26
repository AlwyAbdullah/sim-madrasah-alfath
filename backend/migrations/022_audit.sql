-- 022_audit.sql — pencatatan perubahan: "siapa mengubah terakhir" + lini masa aktivitas.
--
-- Dua lapis yang menjawab dua pertanyaan berbeda:
--   1. updated_by  -> "siapa terakhir mengubah BARIS INI?"  (ditempel di datanya)
--   2. log_aktivitas -> "apa saja yang berubah di sistem?"  (lini masa terpisah)
--
-- Lapis 1 diperlukan karena absensi/nilai/spp disimpan dengan
-- ON DUPLICATE KEY UPDATE, yang TIDAK menyentuh created_by saat baris diperbarui.
-- Jadi selama ini yang tercatat selalu pembuat pertama, bukan pengubah terakhir.
USE sim_madrasah;

-- ---------- Lapis 1: pengubah terakhir ----------
-- SET NULL disamakan dengan created_by: menonaktifkan/menghapus akun tidak boleh
-- menggagalkan penyimpanan data.
ALTER TABLE absensi        ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_absensi_upd    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE nilai          ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_nilai_upd      FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE spp            ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_spp_upd        FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE santri         ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_santri_upd     FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE kelas          ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_kelas_upd      FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE mata_pelajaran ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_mapel_upd      FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE periode        ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_periode_upd    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE guru           ADD COLUMN updated_by BIGINT NULL,
  ADD CONSTRAINT fk_guru_upd       FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;

-- ---------- Lapis 2: lini masa aktivitas ----------
CREATE TABLE IF NOT EXISTS log_aktivitas (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id    BIGINT      NULL,               -- NULL bila akunnya dihapus permanen
  -- username & nama DISALIN, bukan hanya di-join: riwayat harus tetap terbaca
  -- walau akunnya kelak dihapus atau namanya berubah
  username   VARCHAR(50)  NOT NULL,
  nama       VARCHAR(120) NOT NULL,
  aksi       VARCHAR(40)  NOT NULL,          -- mis. 'simpan_absensi', 'reset_password'
  entitas    VARCHAR(40)  NULL,              -- mis. 'absensi', 'santri'
  entitas_id VARCHAR(64)  NULL,
  ringkasan  VARCHAR(500) NOT NULL,          -- teks siap baca manusia
  rincian    JSON         NULL,              -- sebelum -> sesudah, bila perlu ditelusuri
  ip         VARCHAR(64)  NULL,
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_log_waktu (created_at),
  KEY idx_log_user (user_id, created_at),
  KEY idx_log_aksi (aksi, created_at),
  CONSTRAINT fk_log_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
