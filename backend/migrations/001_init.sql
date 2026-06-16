-- SIM-Madrasah — skema awal
-- Jalankan: mysql -u root -p < migrations/001_init.sql
CREATE DATABASE IF NOT EXISTS sim_madrasah
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE sim_madrasah;

CREATE TABLE IF NOT EXISTS users (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(50)  NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  nama          VARCHAR(120) NOT NULL,
  role          ENUM('admin','guru','kepala') NOT NULL DEFAULT 'guru',
  is_active     TINYINT(1)   NOT NULL DEFAULT 1,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS periode (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  nama         VARCHAR(50) NOT NULL,
  tahun_ajaran VARCHAR(20) NOT NULL,
  semester     ENUM('ganjil','genap') NOT NULL,
  is_active    TINYINT(1) NOT NULL DEFAULT 0,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_periode (tahun_ajaran, semester)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS kelas (
  id        BIGINT PRIMARY KEY AUTO_INCREMENT,
  nama      VARCHAR(50) NOT NULL,           -- bebas karakter, mis. "4A", "4B"
  tingkat   VARCHAR(20),
  wali_id   BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_kelas_nama (nama),
  CONSTRAINT fk_kelas_wali FOREIGN KEY (wali_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS santri (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  nis           VARCHAR(30) UNIQUE,
  nama          VARCHAR(120) NOT NULL,
  jenis_kelamin ENUM('L','P') NOT NULL,
  kelas_id      BIGINT NOT NULL,
  is_active     TINYINT(1) NOT NULL DEFAULT 1,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_santri_kelas FOREIGN KEY (kelas_id) REFERENCES kelas(id),
  INDEX idx_santri_kelas (kelas_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS mata_pelajaran (
  id   BIGINT PRIMARY KEY AUTO_INCREMENT,
  kode VARCHAR(20) UNIQUE,
  nama VARCHAR(120) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS absensi (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id   BIGINT NOT NULL,
  tanggal     DATE   NOT NULL,
  status      ENUM('hadir','izin','sakit','alpha') NOT NULL,
  keterangan  VARCHAR(255) NULL,
  created_by  BIGINT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_absensi (santri_id, tanggal),
  CONSTRAINT fk_absensi_santri FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  CONSTRAINT fk_absensi_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  INDEX idx_absensi_tanggal (tanggal)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS nilai (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id         BIGINT NOT NULL,
  mata_pelajaran_id BIGINT NOT NULL,
  periode_id        BIGINT NOT NULL,
  tugas             DECIMAL(5,2) NULL,
  uts               DECIMAL(5,2) NULL,
  uas               DECIMAL(5,2) NULL,
  nilai_akhir       DECIMAL(5,2) NULL,
  created_by        BIGINT NULL,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_nilai (santri_id, mata_pelajaran_id, periode_id),
  CONSTRAINT fk_nilai_santri FOREIGN KEY (santri_id) REFERENCES santri(id) ON DELETE CASCADE,
  CONSTRAINT fk_nilai_mapel FOREIGN KEY (mata_pelajaran_id) REFERENCES mata_pelajaran(id),
  CONSTRAINT fk_nilai_periode FOREIGN KEY (periode_id) REFERENCES periode(id),
  CONSTRAINT fk_nilai_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
