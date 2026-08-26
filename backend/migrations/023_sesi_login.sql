-- 023_sesi_login.sql — pelacakan sesi login.
--
-- JWT bersifat stateless: server tidak tahu siapa yang sedang masuk, dan logout
-- hanya menghapus cookie sementara tokennya tetap sah sampai kedaluwarsa
-- (120 menit). Akibatnya menonaktifkan akun TIDAK langsung memutus orangnya.
--
-- Tabel ini menambahkan state secukupnya untuk:
--   1. menampilkan siapa yang sedang aktif,
--   2. membuat logout benar-benar mencabut token,
--   3. memungkinkan superadmin memutus sesi orang lain.
--
-- Biayanya satu pembacaan ber-indeks per permintaan — tidak terasa untuk
-- madrasah dengan 20 pengguna, dan sepadan selama password awal masih seragam.
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS sesi_login (
  id             CHAR(36)     PRIMARY KEY,          -- juga ditanam di JWT (klaim sid)
  user_id        BIGINT       NOT NULL,
  ip             VARCHAR(64)  NULL,
  user_agent     VARCHAR(255) NULL,
  dibuat_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  terakhir_aktif DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  -- terisi saat logout / diputus superadmin; sesi dengan nilai ini TIDAK berlaku lagi
  berakhir_at    DATETIME     NULL,
  -- siapa yang memutus (NULL = logout sendiri)
  diputus_oleh   BIGINT       NULL,
  KEY idx_sesi_user (user_id, berakhir_at),
  KEY idx_sesi_aktif (berakhir_at, terakhir_aktif),
  CONSTRAINT fk_sesi_user    FOREIGN KEY (user_id)      REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_sesi_pemutus FOREIGN KEY (diputus_oleh) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
