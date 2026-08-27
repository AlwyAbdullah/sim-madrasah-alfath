-- 024_telegram_pribadi.sql — notifikasi Telegram per orang.
--
-- Sampai sekarang pengingat absensi hanya dikirim ke SATU grup, sehingga
-- "Kelas 3 belum diabsen" dibaca 20 orang dan tidak ada yang merasa itu
-- tugasnya. Dengan penautan akun ke Telegram, pengingat bisa tertuju:
-- wali kelas menerima kabar kelasnya sendiri, superadmin menerima ringkasan.
--
-- users.telegram_user_id sudah ada sejak migrasi 007 (dan belum pernah dipakai);
-- yang kurang hanya cara menautkannya tanpa harus menyalin id numerik manual.
USE sim_madrasah;

ALTER TABLE users
  -- kode sekali pakai yang dikirim pengguna ke bot untuk membuktikan chat itu miliknya
  ADD COLUMN telegram_kode     VARCHAR(16)  NULL,
  ADD COLUMN telegram_kode_exp DATETIME     NULL,
  -- nama/username Telegram, hanya untuk ditampilkan agar penggunanya yakin
  -- akun yang tertaut memang miliknya
  ADD COLUMN telegram_nama     VARCHAR(120) NULL,
  ADD UNIQUE KEY uq_users_tg_kode (telegram_kode);

ALTER TABLE telegram_pengaturan
  -- penanda pesan terakhir yang sudah diproses; tanpa ini pesan yang sama
  -- terbaca berulang setiap polling
  ADD COLUMN update_offset BIGINT      NOT NULL DEFAULT 0,
  -- diisi otomatis dari getMe, dipakai menyusun tautan https://t.me/<bot>?start=<kode>
  ADD COLUMN bot_username  VARCHAR(64) NULL;

-- Bersihkan isi lama telegram_user_id.
--
-- Kolom ini ada sejak migrasi 007 tapi tidak pernah dipakai fitur apa pun, dan
-- di produksi berisi nilai contoh `123` pada akun admin. Mulai sekarang kolom
-- itu BENAR-BENAR dipakai sebagai tujuan pengiriman, sehingga nilai karangan
-- akan membuat pengingat superadmin gagal terkirim setiap hari sekolah dan
-- memenuhi antrean dengan kegagalan.
--
-- Aman dikosongkan: satu-satunya cara sah mengisinya sekarang adalah lewat
-- penautan berkode, yang belum pernah dijalankan siapa pun.
UPDATE users SET telegram_user_id = NULL WHERE telegram_user_id IS NOT NULL;
