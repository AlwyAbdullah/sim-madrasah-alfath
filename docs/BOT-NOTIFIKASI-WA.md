# Notifikasi WhatsApp ke Orang Tua

Backend mengantrikan pesan (tabel `notifikasi_wa`) setiap kali santri ditandai
**alpha**. Ada **dua cara** mengirimkannya — pilih salah satu:

| Cara | Kapan cocok |
|---|---|
| **A. Worker bawaan** (bagian ini) | Paling sederhana — cukup isi beberapa baris `.env`, tanpa n8n/skrip tambahan |
| **B. n8n / skrip / bot kustom** | Kalau pengiriman WA sudah dikelola tim/alur lain (lihat bagian bawah + `N8N-NOTIFIKASI-WA.md`) |

Keduanya memakai antrean yang **sama** — mengaktifkan cara A tidak merusak cara B,
dan sebaliknya. Kalau `WAHA_URL` kosong, worker bawaan diam saja dan cara B tetap
berfungsi normal.

```
Guru simpan absensi (alpha)
        ↓
Backend antrikan pesan  →  tabel notifikasi_wa (status = pending)
        ↓
   ┌─────────────────────────┴─────────────────────────┐
   │ A. Worker bawaan (goroutine di dalam backend)      │
   │    - polling tiap WAHA_POLL_SECONDS                │
   │    - POST langsung ke {WAHA_URL}/api/sendText      │
   │    - update status sendiri, tanpa panggilan API    │
   │                                                     │
   │ B. Bot/n8n eksternal                               │
   │    - GET  /api/v1/notifikasi/pending                │
   │    - POST /api/v1/notifikasi/status                 │
   └─────────────────────────────────────────────────────┘
```

Kalau proses pengirimnya (worker atau bot) sedang mati, pesan **tetap aman
menunggu** di antrean — tidak ada yang hilang.

---

## A. Worker bawaan (rekomendasi)

Isi variabel berikut di `.env` backend (VPS), lalu **restart service backend**:

```bash
WAHA_URL=http://alamat-waha-anda:3000   # kosongkan untuk menonaktifkan worker
WAHA_SESSION=default                     # nama sesi WAHA Anda
WAHA_API_KEY=                            # isi bila WAHA Anda memakai API key
WAHA_POLL_SECONDS=60                     # jeda antar putaran polling
WAHA_SEND_DELAY_SECONDS=8                # jeda antar pesan (anti-spam)
WAHA_BATCH_LIMIT=20                      # maksimum pesan per putaran
```

```bash
sudo systemctl restart sim-madrasah-backend
journalctl -u sim-madrasah-backend -f     # pantau log worker
```

Log yang muncul saat aktif:

```
notifworker: aktif — polling tiap 1m0s, kirim ke http://alamat-waha-anda:3000 (sesi "default")
notifworker: terkirim id=12 ke 6281234567890
```

Bila `WAHA_URL` kosong, log-nya:

```
notifworker: WAHA_URL kosong -> worker pengirim WA tidak aktif (pakai n8n/skrip lain bila perlu)
```

**Tidak perlu instal apa pun tambahan** — cukup backend yang sudah berjalan.
Sudah diuji end-to-end (lokal, dengan WAHA tiruan): pengiriman sukses, pengiriman
gagal (alasannya tercatat), dan pemrosesan berjalan otomatis tanpa campur tangan
manual.

### Catatan perilaku "Kirim Ulang"

Tombol **Kirim ulang** di halaman admin mengirim ulang **pesan & nomor yang sama**
persis seperti saat pertama diantrekan — bukan mengambil nomor terbaru dari data
santri. Jadi kalau pesan gagal **karena nomor orang tua salah**, memperbaiki nomor
di Master Santri lalu menekan "Kirim ulang" **tidak** memakai nomor baru itu.
Gunakan "Kirim ulang" untuk kegagalan sementara (mis. WAHA sedang mati); untuk
nomor yang salah, tunggu kejadian alpha berikutnya (akan mengantre dengan nomor
terbaru) atau perbarui langsung baris di tabel `notifikasi_wa`.

---

## B. Bot / n8n eksternal (alternatif)

Kalau tidak memakai worker bawaan, proses eksternal Anda memanggil dua endpoint
berikut. Autentikasi memakai `BOT_SHARED_SECRET` yang sama dengan `/auth/bot-login`:

```
X-Bot-Secret: <BOT_SHARED_SECRET>
```

Base URL produksi: `https://madrasah-alfath-malang.web.id/api/v1`

### 1. Ambil antrean

```http
GET /api/v1/notifikasi/pending?limit=20
X-Bot-Secret: <secret>
```

Jawaban:

```json
{
  "items": [
    {
      "id": 12,
      "tujuan": "6281234567890",
      "pesan": "Assalamu'alaikum ...",
      "jenis": "absensi_alpha"
    }
  ]
}
```

- `tujuan` sudah **dinormalkan** ke format `628xxxxxxxxx` (siap dipakai; tambahkan
  sendiri suffix `@c.us` bila library Anda memerlukannya).
- `pesan` sudah jadi teks final (mengandung `\n` dan penebalan gaya WhatsApp `*teks*`).
- `limit` opsional, default 20, maksimum 100.

### 2. Lapor hasil pengiriman

```http
POST /api/v1/notifikasi/status
X-Bot-Secret: <secret>
Content-Type: application/json

{ "id": 12, "sukses": true }
```

Bila gagal, sertakan alasannya supaya muncul di halaman admin:

```json
{ "id": 12, "sukses": false, "catatan": "nomor tidak terdaftar di WhatsApp" }
```

Pesan yang gagal **tidak** otomatis dicoba ulang — admin bisa menekan
**"Kirim ulang"** di halaman *Notifikasi WA*, dan pesan kembali menjadi `pending`
(dengan catatan yang sama seperti dijelaskan di bagian A).

### Contoh polling (Node.js)

```js
const BASE = "https://madrasah-alfath-malang.web.id/api/v1";
const SECRET = process.env.BOT_SHARED_SECRET;
const H = { "X-Bot-Secret": SECRET, "Content-Type": "application/json" };

async function prosesAntrean() {
  const r = await fetch(`${BASE}/notifikasi/pending?limit=20`, { headers: H });
  const { items } = await r.json();

  for (const it of items) {
    try {
      await kirimWhatsApp(`${it.tujuan}@c.us`, it.pesan);   // fungsi bot Anda
      await fetch(`${BASE}/notifikasi/status`, {
        method: "POST", headers: H,
        body: JSON.stringify({ id: it.id, sukses: true }),
      });
    } catch (e) {
      await fetch(`${BASE}/notifikasi/status`, {
        method: "POST", headers: H,
        body: JSON.stringify({ id: it.id, sukses: false, catatan: String(e).slice(0, 200) }),
      });
    }
    await new Promise((r) => setTimeout(r, 3000)); // jeda antar pesan
  }
}

setInterval(prosesAntrean, 60_000); // cek tiap 1 menit
```

Untuk versi n8n siap impor, lihat **`N8N-NOTIFIKASI-WA.md`** dan
**`n8n-notifikasi-wa.json`** di folder ini.

---

## Catatan perilaku (berlaku untuk cara A maupun B)

| Kejadian | Yang terjadi di antrean |
|---|---|
| Santri ditandai **alpha** | Pesan dibuat, `status = pending` |
| Status dikoreksi jadi hadir/izin/sakit | Pesan yang **belum terkirim** otomatis jadi `batal` |
| Absensi disimpan ulang di hari sama | **Tidak** membuat duplikat (unik per santri + tanggal) |
| Sudah `terkirim` lalu absensi disimpan lagi | Tetap `terkirim` — tidak dikirim dua kali |
| Nomor orang tua kosong / tidak valid | Dilewati diam-diam; absensi tetap tersimpan |

Hanya status **alpha** yang memicu notifikasi (izin & sakit tidak).

## Pantau dari aplikasi

Halaman **Kepegawaian → 💬 Notifikasi WA** (khusus admin) menampilkan seluruh
antrean beserta status, isi pesan, jumlah percobaan, alasan gagal, serta tombol
**Kirim ulang** dan **Batal** — berlaku sama untuk pesan yang diproses lewat
worker bawaan maupun lewat bot/n8n eksternal.
