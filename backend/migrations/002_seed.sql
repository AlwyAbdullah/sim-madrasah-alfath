-- SIM-Madrasah — data contoh
-- Jalankan setelah 001_init.sql: mysql -u root -p < migrations/002_seed.sql
USE sim_madrasah;

-- User admin awal. Username: admin  Password: admin123
INSERT INTO users (username, password_hash, nama, role) VALUES
  ('admin', '$2a$12$Jm3nXtsm.eLv9U7hU.cFT.u9MadTgDwIswKslZyijQUYhetB1EiwS', 'Administrator', 'admin'),
  ('guru', '$2a$12$Jm3nXtsm.eLv9U7hU.cFT.u9MadTgDwIswKslZyijQUYhetB1EiwS', 'Ustadz Contoh', 'guru')
ON DUPLICATE KEY UPDATE nama = VALUES(nama);

INSERT INTO periode (nama, tahun_ajaran, semester, is_active) VALUES
  ('2025/2026 - Ganjil', '2025/2026', 'ganjil', 1)
ON DUPLICATE KEY UPDATE is_active = VALUES(is_active);

INSERT INTO kelas (nama, tingkat) VALUES
  ('7A', '7'), ('7B', '7'), ('8A', '8')
ON DUPLICATE KEY UPDATE tingkat = VALUES(tingkat);

INSERT INTO mata_pelajaran (kode, nama) VALUES
  ('FQH', 'Fiqih'), ('AQD', 'Aqidah Akhlak'), ('QHD', 'Quran Hadits'),
  ('SKI', 'Sejarah Kebudayaan Islam'), ('BAR', 'Bahasa Arab'), ('MTK', 'Matematika')
ON DUPLICATE KEY UPDATE nama = VALUES(nama);

INSERT INTO santri (nis, nama, jenis_kelamin, kelas_id) VALUES
  ('2025001', 'Ahmad Fauzan',   'L', 1),
  ('2025002', 'Bilal Ramadhan', 'L', 1),
  ('2025003', 'Citra Aulia',    'P', 1),
  ('2025004', 'Dina Salsabila', 'P', 1),
  ('2025005', 'Erlangga Putra', 'L', 2),
  ('2025006', 'Fatimah Zahra',  'P', 2),
  ('2025007', 'Gilang Saputra', 'L', 3),
  ('2025008', 'Hana Maharani',  'P', 3)
ON DUPLICATE KEY UPDATE nama = VALUES(nama);
