# Panduan n8n — Mengirim Notifikasi WA ke Orang Tua

> 💡 **Tidak wajib pakai n8n.** Backend punya **worker bawaan** yang bisa
> mengirim langsung ke WAHA tanpa n8n — cukup isi `WAHA_URL` dkk di `.env`.
> Lihat bagian "A. Worker bawaan" di `BOT-NOTIFIKASI-WA.md`. Pakai panduan n8n
> ini hanya bila Anda memang ingin pengiriman dikelola lewat n8n (mis. supaya
> mudah dipantau visual, atau digabung dengan automasi lain).

Backend hanya **mengantrikan** pesan. Workflow n8n ini yang mengambil antrean,
mengirim lewat **WAHA**, lalu melapor balik. Tidak perlu mengubah kode bot yang ada.

```
Schedule (5 mnt) → GET antrean → per pesan: WAHA sendText → lapor status → jeda
```

## Yang perlu disiapkan

| Item | Nilai |
|---|---|
| Base URL API | `https://madrasah-alfath-malang.web.id/api/v1` |
| Header autentikasi | `X-Bot-Secret: <BOT_SHARED_SECRET>` (sama dengan yang dipakai bot-login) |
| URL WAHA | mis. `http://waha:3000` atau alamat instance WAHA Anda |
| Session WAHA | biasanya `default` |

> 🔐 Simpan `BOT_SHARED_SECRET` dan API key WAHA sebagai **Credential** di n8n
> (Header Auth), jangan ditulis langsung di node.

> 📱 **Disarankan pakai nomor WA terpisah** untuk notifikasi orang tua — supaya
> kalau nomor kena pembatasan, input data guru lewat WA tidak ikut terganggu.

---

## Langkah membuat workflow (manual)

### 1. Schedule Trigger
- Tipe: **Schedule Trigger**
- Interval: **Minutes**, setiap **5** menit

### 2. Ambil antrean — HTTP Request
- Nama: `Ambil Antrean`
- Method: **GET**
- URL: `https://madrasah-alfath-malang.web.id/api/v1/notifikasi/pending?limit=20`
- Headers: `X-Bot-Secret` = `<secret>`
- Response: JSON

Hasil: `{ "items": [ { id, tujuan, pesan, jenis } ] }`

### 3. Pecah array — Split Out
- Tipe: **Split Out** (n8n lama: *Item Lists → Split Out Items*)
- Field to split out: `items`

Kalau antrean kosong, workflow berhenti di sini dengan sendirinya.

### 4. Proses satu per satu — Loop Antrean
- Tipe: **Loop Over Items** (*Split In Batches*) — beri nama node `Loop Antrean`
- Batch Size: **1**

### 5. Kirim via WAHA — HTTP Request
- Nama: `Kirim WAHA`
- Method: **POST**
- URL: `{WAHA_URL}/api/sendText`
- Headers: `X-Api-Key` = `<api key WAHA>` (bila WAHA Anda memakainya)
- Body (JSON):

```json
{
  "session": "default",
  "chatId": "={{ $json.tujuan }}@c.us",
  "text": "={{ $json.pesan }}"
}
```

> `tujuan` sudah dinormalkan backend ke `628xxxxxxxxx`, tinggal ditambah `@c.us`.

**Penting:** aktifkan **Settings → On Error → Continue (using error output)** pada
node ini, supaya satu nomor yang gagal tidak menghentikan sisa antrean.

### 6. Lapor berhasil — HTTP Request
- Method: **POST**
- URL: `https://madrasah-alfath-malang.web.id/api/v1/notifikasi/status`
- Headers: `X-Bot-Secret` = `<secret>`
- Body (JSON):

```json
{ "id": "={{ $('Loop Antrean').item.json.id }}", "sukses": true }
```

### 7. Lapor gagal — HTTP Request (dari jalur error node 5)
Body:

```json
{
  "id": "={{ $('Loop Antrean').item.json.id }}",
  "sukses": false,
  "catatan": "={{ $json.error || 'gagal kirim' }}"
}
```

### 8. Jeda — Wait
- Tipe: **Wait**
- Amount: **8** detik (rentang aman 5–15 detik)
- Sambungkan kembali ke **Loop Antrean**

---

## Cara cepat: impor file JSON

File siap impor: **[`n8n-notifikasi-wa.json`](n8n-notifikasi-wa.json)**

1. Di n8n: **Workflows → ⋯ (titik tiga) → Import from File** lalu pilih file tersebut.
2. Ganti 3 placeholder ini (Ctrl+F di tiap node):

   | Placeholder | Isi dengan |
   |---|---|
   | `__BOT_SECRET__` | nilai `BOT_SHARED_SECRET` (2 node: *Lapor Sukses* & *Lapor Gagal*, dan *Ambil Antrean*) |
   | `__WAHA_URL__` | alamat WAHA Anda, mis. `http://waha:3000` (tanpa garis miring di akhir) |
   | `__WAHA_API_KEY__` | API key WAHA — **hapus baris header ini** bila WAHA Anda tanpa API key |

3. Sesuaikan `session` di node *Kirim WAHA* bila bukan `default`.
4. **Execute Workflow** untuk uji manual, lalu **Active** bila sudah benar.

> ⚠️ Setelah placeholder diisi, file berisi rahasia — **jangan** di-commit ke git
> atau dibagikan. Lebih aman lagi: simpan sebagai **Credential** *Header Auth* di n8n.

Struktur node bisa sedikit berbeda antar versi n8n — sesuaikan bila perlu.

### Alur workflow

```
        ┌──────────────────┐
        │ Setiap 5 Menit   │  (Schedule Trigger)
        └────────┬─────────┘
                 ▼
        ┌──────────────────┐
        │ Ambil Antrean    │  GET /notifikasi/pending?limit=20
        └────────┬─────────┘  header: X-Bot-Secret
                 ▼
        ┌──────────────────┐
        │ Pecah Items      │  Split Out: field "items"
        └────────┬─────────┘
                 ▼
        ┌──────────────────┐
   ┌───▶│ Loop Antrean     │  Split In Batches (1 per putaran)
   │    └────────┬─────────┘
   │             ▼ (output "loop")
   │    ┌──────────────────┐
   │    │ Kirim WAHA       │  POST {WAHA}/api/sendText
   │    └───┬──────────┬───┘  onError: continue (error output)
   │  sukses▼          ▼gagal
   │  ┌───────────┐ ┌───────────┐
   │  │Lapor Sukses│ │Lapor Gagal│  POST /notifikasi/status
   │  └─────┬─────┘ └─────┬─────┘
   │        └──────┬──────┘
   │               ▼
   │      ┌──────────────────┐
   └──────┤ Jeda 8 Detik     │  Wait (anti-spam)
          └──────────────────┘
```

Isi lengkap tiap node ada di file **[`n8n-notifikasi-wa.json`](n8n-notifikasi-wa.json)**.

---

## Cara menguji tanpa mengganggu orang tua

1. Isi **nomor sendiri** pada kolom *No. Orang Tua* satu santri (Master Data → Santri).
2. Tandai santri itu **alpha** di halaman Absensi, lalu simpan.
3. Buka **Kepegawaian → 💬 Notifikasi WA** — pesan muncul dengan status *Menunggu*.
4. Jalankan workflow n8n secara manual (**Execute Workflow**).
5. Pesan masuk ke WA Anda, dan status di halaman berubah jadi **Terkirim**.
6. Kembalikan nomor orang tua seperti semula.

## Bila ada yang gagal

- Status berubah **Gagal** dan alasannya tampil di halaman Notifikasi WA.
- Perbaiki penyebabnya (mis. nomor salah), lalu tekan **Kirim ulang** — pesan
  kembali ke antrean dan akan diambil workflow berikutnya.

## Agar nomor tidak diblokir WhatsApp

- Pertahankan jeda **5–15 detik** antar pesan (node *Wait*).
- Jangan naikkan `limit` terlalu tinggi; **20 per putaran** sudah cukup.
- Minta orang tua **menyimpan nomor madrasah** di kontak mereka.
- Gunakan **nomor khusus** untuk notifikasi, terpisah dari nomor bot guru.
- Untuk kebutuhan besar & bebas risiko, pertimbangkan **WhatsApp Business API resmi**.
