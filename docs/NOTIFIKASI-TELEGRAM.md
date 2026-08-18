# Notifikasi Telegram

Telegram adalah **satu-satunya** kanal notifikasi SIM Madrasah. WhatsApp sudah dihapus
seluruhnya (lihat bagian terakhir).

## Cara kerja

Backend tidak pernah memanggil Telegram langsung dari alur kerja pengguna. Semua pesan
masuk dulu ke tabel antrean `notifikasi` berstatus `pending`, lalu worker
`internal/notifworker` mengambil dan mengirimnya tiap `NOTIF_POLL_SECONDS` (bawaan 60 detik).

Konsekuensinya yang berguna: bila pengiriman sedang dijeda dari halaman admin, atau Telegram
sedang tidak bisa dihubungi, pesan **tidak hilang** — tetap tercatat dan terkirim sendiri
begitu keadaan normal.

```
pengingat/pemicu → INSERT ke `notifikasi` (pending)
                        ↓  worker polling
                   Telegram Bot API → status 'terkirim' / 'gagal' + catatan
```

## Pemasangan

1. **Token bot** — buat lewat `@BotFather` (`/newbot`), lalu pasang di `.env` backend:
   ```
   TELEGRAM_BOT_TOKEN=123456:AA...
   ```
   File `.env` produksi wajib `chmod 600`. Token **tidak pernah** dikirim ke browser —
   halaman admin hanya menampilkan status "sudah terpasang / belum".

2. **Tujuan chat** — diatur dari halaman admin **Notifikasi**, bukan dari `.env`.
   Cara memperoleh `chat_id`:
   - **Grup guru (disarankan):** tambahkan bot ke grup, kirim satu pesan apa saja di grup.
   - **Chat pribadi:** buka `https://t.me/<username_bot>`, tekan **START**.

   Lalu baca `chat_id`-nya:
   ```bash
   curl -s "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getUpdates"
   ```
   Isikan nilainya di halaman Notifikasi, lalu tekan **Kirim uji** untuk memastikan sambungan
   benar sebelum diandalkan.

> ⚠️ **Grup biasa vs supergroup.** Telegram mengganti `chat_id` grup saat grup naik menjadi
> supergroup (terjadi otomatis, mis. saat grup dijadikan publik atau anggotanya bertambah
> banyak). Bila itu terjadi, pengiriman mulai `gagal` dengan catatan semacam
> *"group chat was upgraded to a supergroup chat"* — ambil `chat_id` baru lewat `getUpdates`
> dan perbarui dari halaman Notifikasi. Tidak perlu ubah kode.

## Pengingat yang berjalan

| Pengingat | Kapan | Isi |
|---|---|---|
| **Absensi harian** | tiap hari sekolah pada jam yang diatur (bawaan 19:00) | kelas yang belum / baru sebagian diabsen |
| **SPP bulanan** | tiap tanggal yang diatur (bawaan tanggal 10, 19:00) | santri yang belum membayar SPP **bulan berjalan**, dikelompokkan per kelas |

Keduanya punya sifat yang sama:

- **Jadwalnya di database, bukan di kode** (`pengingat_absensi_pengaturan`,
  `pengingat_spp_pengaturan`) sehingga admin bisa menggesernya tanpa deploy ulang.
- **Tahan mati listrik.** Pemeriksaan memakai perbandingan `>=`, bukan "tepat menit ini",
  jadi pengingat tetap terkirim walau server sempat mati saat jamnya lewat.
- **Tidak berisik.** Bila tidak ada yang perlu diingatkan (semua kelas sudah diabsen / semua
  santri sudah lunas), tidak ada pesan sama sekali — hanya ditandai supaya tidak diperiksa
  berulang.
- **Sekali per periode.** Absensi dikunci per tanggal, SPP per bulan (`terakhir_kirim`).

### Catatan khusus pengingat SPP

- Yang dihitung "belum bayar" mencakup santri yang **belum punya baris SPP sama sekali**,
  bukan hanya yang barisnya `lunas = 0`. Ini disengaja: di produksi banyak santri memang
  belum punya barisnya, dan mereka jelas belum membayar.
- Tanggal pengiriman dibatasi **1–28** agar selalu ada di setiap bulan (Februari tidak
  punya tanggal 29–31).
- Bila daftar nama melebihi satu pesan Telegram (batas 4096 karakter), pesan otomatis
  disusun ulang tanpa nama — hanya jumlah per kelas — dengan catatan agar membuka halaman SPP.
  Lebih baik ringkas tapi utuh daripada terpotong di tengah nama.
- Tombol **Kirim sekarang** di halaman Notifikasi mengirim di luar jadwal dan sengaja
  **tidak** mengubah `terakhir_kirim`, sehingga tidak membatalkan pengingat terjadwal bulan itu.

## Pemecahan masalah

Semua percobaan pengiriman tercatat di halaman **Notifikasi** lengkap dengan pesan galat
apa adanya dari Telegram.

| Catatan galat | Artinya |
|---|---|
| `chat not found` | bot belum ditambahkan ke grup, atau belum ditekan START di chat pribadi |
| `bot was kicked from the group` | bot dikeluarkan dari grup — masukkan lagi |
| `group chat was upgraded to a supergroup` | `chat_id` berubah, lihat peringatan di atas |
| `Telegram balas 401` | token salah / sudah dicabut lewat `@BotFather` |

Pesan berstatus `gagal` bisa dikembalikan ke antrean dengan tombol **Kirim ulang**.

Log worker:
```bash
sudo journalctl -u sim-madrasah-backend -e | grep -E "telegram|pengingat"
```

## Tentang WhatsApp

WhatsApp pernah dipakai lewat **WAHA** (pembungkus Baileys — API **tidak resmi**). Nomor yang
dipakai sempat dibatasi Meta (`device_removed` + `RESTRICT_ALL_COMPANIONS`), yang memang risiko
bawaan cara itu.

Sejak migrasi `018_hapus_whatsapp.sql`, WhatsApp dihapus **seluruhnya**, bukan sekadar dimatikan:

- container WAHA dibuang dari VPS;
- paket `internal/waid`, worker WAHA, dan endpoint `/auth/bot-login`, `/notifikasi/pending`,
  `/notifikasi/status` dihapus;
- kolom `users.whatsapp_number` dan kolom `notifikasi.kanal` dihapus;
- tabel `notifikasi_wa` → `notifikasi`, `notifikasi_wa_pengaturan` → `notifikasi_pengaturan`.

Notifikasi alpha ke orang tua ikut dihapus bersamanya: fitur itu bergantung pada
`santri.no_ortu`, dan kolom tersebut kosong untuk **seluruh** santri aktif di produksi,
sehingga tidak pernah benar-benar mengirim apa pun. Kolom `no_ortu` sendiri dipertahankan
karena isinya data kontak biasa. Bila suatu saat orang tua ingin diberi notifikasi, jalur yang
layak adalah **WhatsApp Business API resmi** (berbayar, lewat penyedia terdaftar) — bukan
menghidupkan kembali WAHA.
