-- 018_hapus_whatsapp.sql — hapus seluruh jejak WhatsApp dari sistem.
--
-- Alasan: WhatsApp hanya bisa dipakai lewat API tidak resmi (WAHA/Baileys),
-- dan nomornya memang sempat dibatasi ('device_removed' + RESTRICT_ALL_COMPANIONS).
-- Telegram Bot API sudah terbukti jalan dan resmi, jadi WhatsApp tidak lagi dipakai
-- sama sekali — bukan sekadar dimatikan.
--
-- Catatan tentang notifikasi alpha ke orang tua: fitur itu bergantung pada
-- santri.no_ortu, dan di produksi kolom itu KOSONG untuk seluruh 100 santri aktif,
-- sehingga tidak pernah benar-benar mengirim apa pun. Kolom no_ortu SENGAJA
-- DIPERTAHANKAN karena isinya data kontak biasa (nomor HP orang tua), bukan
-- sesuatu yang khusus WhatsApp.
USE sim_madrasah;

-- 1) Antrean WhatsApp yang belum terkirim tidak akan pernah bisa dikirim lagi.
--    Ditandai 'batal' (bukan dihapus) supaya riwayatnya tetap terbaca.
UPDATE notifikasi_wa
   SET status = 'batal', catatan = 'kanal WhatsApp dihentikan (migrasi 018)'
 WHERE kanal = 'whatsapp' AND status IN ('pending', 'gagal');

-- 2) Nama tabel tidak lagi menyebut WhatsApp. Baris lama tetap utuh.
RENAME TABLE notifikasi_wa            TO notifikasi;
RENAME TABLE notifikasi_wa_pengaturan TO notifikasi_pengaturan;

-- 3) Hanya tersisa satu kanal (Telegram), jadi kolom pembeda kanal tidak berguna lagi.
ALTER TABLE notifikasi DROP COLUMN kanal;

-- 4) Sesuaikan kolom untuk Telegram:
--    - tujuan: chat_id Telegram bisa 14 karakter (grup: -100xxxxxxxxxx), bukan
--      nomor HP 13 karakter. Dilebarkan agar aman bila grup naik jadi supergroup.
--    - pesan: rekap tunggakan SPP satu madrasah jauh lebih panjang daripada satu
--      pesan alpha. Batas Telegram 4096 karakter; kolom disamakan.
ALTER TABLE notifikasi
  MODIFY COLUMN tujuan VARCHAR(64)   NOT NULL,
  MODIFY COLUMN pesan  VARCHAR(4000) NOT NULL;

-- 5) Nomor WhatsApp guru dipakai hanya oleh /auth/bot-login (bot WhatsApp), yang
--    ikut dihapus. Di produksi hanya 1 baris terisi (akun admin).
ALTER TABLE users DROP COLUMN whatsapp_number;
