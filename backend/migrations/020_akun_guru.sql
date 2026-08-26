-- 020_akun_guru.sql — akun login per guru + peran superadmin.
--
-- Latar belakang: sebelum ini seluruh madrasah memakai 2 akun (`admin`, `guru`)
-- untuk 20 guru, sehingga kolom created_by di 1.489 baris absensi hanya pernah
-- berisi 1 atau 2 — audit trail-nya ada tapi tidak bermakna.
--
-- Akun-akunnya TIDAK dibuat di sini: password wajib di-hash bcrypt dan itu tidak
-- bisa dilakukan di dalam .sql. Pembuatannya lewat POST /users/dari-guru.
USE sim_madrasah;

-- superadmin = admin + boleh mengelola akun admin/superadmin lain, mereset
-- password, dan memutus sesi. 'kepala' dibiarkan ada walau tidak dipakai.
ALTER TABLE users
  MODIFY COLUMN role ENUM('superadmin','admin','guru','kepala') NOT NULL DEFAULT 'guru';

ALTER TABLE users
  ADD COLUMN guru_id BIGINT NULL AFTER role,
  ADD COLUMN terakhir_login DATETIME NULL;

-- Satu guru paling banyak satu akun. FK-nya SET NULL supaya menghapus baris guru
-- tidak ikut menghapus akunnya (akun masih dipakai sebagai referensi created_by).
ALTER TABLE users
  ADD CONSTRAINT uq_users_guru UNIQUE (guru_id),
  ADD CONSTRAINT fk_users_guru FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE SET NULL;

-- Ejaan nama guru #2 dirapikan agar sesuai nama sebenarnya
-- ("Ustadz Idrus Tsani bin Agil"), bukan sekadar "SANI BIN AGIL".
UPDATE guru SET nama = 'IDRUS TSANI BIN AGIL'
 WHERE id = 2 AND TRIM(nama) = 'SANI BIN AGIL';

-- Akun `admin` yang ada dinaikkan jadi superadmin. Ini perlu supaya ada yang
-- BISA membuat akun superadmin untuk Muhammad Al Masyhur, Alwy Alaydrus, dan
-- Sholeh Assegaf — tanpa langkah ini tidak ada satu pun akun yang berwenang
-- melakukannya, dan sistem terkunci pada dirinya sendiri.
-- Setelah ketiga akun itu dibagikan, akun `admin` dinonaktifkan.
UPDATE users SET role = 'superadmin' WHERE username = 'admin';
