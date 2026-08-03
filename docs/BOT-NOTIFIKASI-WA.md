# Integrasi Bot WhatsApp — Notifikasi Orang Tua

Backend **tidak mengirim** pesan WhatsApp sendiri. Backend hanya **mengantrikan**
pesan (outbox), lalu **bot WhatsApp** yang mengambil dan mengirimnya, kemudian
melapor balik hasilnya.

```
Guru simpan absensi (alpha)
        ↓
Backend antrikan pesan  →  tabel notifikasi_wa (status = pending)
        ↓
Bot: GET  /api/v1/notifikasi/pending     ← ambil antrean
Bot: kirim WhatsApp
Bot: POST /api/v1/notifikasi/status      → lapor sukses/gagal
```

Kalau bot sedang mati, pesan **tetap aman menunggu** di antrean.

## Autentikasi

Kedua endpoint memakai `BOT_SHARED_SECRET` yang sama dengan `/auth/bot-login`.
Kirim lewat header (disarankan) atau query string:

```
X-Bot-Secret: <BOT_SHARED_SECRET>
```

Base URL produksi: `https://madrasah-alfath-malang.web.id/api/v1`

## 1. Ambil antrean

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

## 2. Lapor hasil pengiriman

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
**"Kirim ulang"** di halaman *Notifikasi WA*, dan pesan kembali menjadi `pending`.

## Contoh polling (Node.js)

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

**Saran:** beri jeda beberapa detik antar pesan agar nomor tidak dianggap spam.

## Catatan perilaku

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
**Kirim ulang** dan **Batal**.
