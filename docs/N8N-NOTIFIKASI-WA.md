# Panduan n8n — Mengirim Notifikasi WA ke Orang Tua

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

### 4. Proses satu per satu — Loop Over Items
- Tipe: **Loop Over Items** (*Split In Batches*)
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
{ "id": "={{ $('Loop Over Items').item.json.id }}", "sukses": true }
```

### 7. Lapor gagal — HTTP Request (dari jalur error node 5)
Body:

```json
{
  "id": "={{ $('Loop Over Items').item.json.id }}",
  "sukses": false,
  "catatan": "={{ $json.error || 'gagal kirim' }}"
}
```

### 8. Jeda — Wait
- Tipe: **Wait**
- Amount: **8** detik (rentang aman 5–15 detik)
- Sambungkan kembali ke **Loop Over Items**

---

## Kerangka JSON (untuk diimpor)

Ganti `GANTI_WAHA_URL`, lalu isi credential setelah impor. Struktur node bisa
sedikit berbeda antar versi n8n — sesuaikan bila perlu.

```json
{
  "name": "Notifikasi WA Madrasah Al Fath",
  "nodes": [
    { "name": "Setiap 5 Menit", "type": "n8n-nodes-base.scheduleTrigger", "typeVersion": 1.2,
      "position": [0, 0],
      "parameters": { "rule": { "interval": [ { "field": "minutes", "minutesInterval": 5 } ] } } },

    { "name": "Ambil Antrean", "type": "n8n-nodes-base.httpRequest", "typeVersion": 4.2,
      "position": [220, 0],
      "parameters": {
        "url": "https://madrasah-alfath-malang.web.id/api/v1/notifikasi/pending?limit=20",
        "sendHeaders": true,
        "headerParameters": { "parameters": [ { "name": "X-Bot-Secret", "value": "GANTI_SECRET" } ] }
      } },

    { "name": "Pecah Items", "type": "n8n-nodes-base.splitOut", "typeVersion": 1,
      "position": [440, 0], "parameters": { "fieldToSplitOut": "items" } },

    { "name": "Loop Over Items", "type": "n8n-nodes-base.splitInBatches", "typeVersion": 3,
      "position": [660, 0], "parameters": { "batchSize": 1 } },

    { "name": "Kirim WAHA", "type": "n8n-nodes-base.httpRequest", "typeVersion": 4.2,
      "position": [880, 0], "onError": "continueErrorOutput",
      "parameters": {
        "method": "POST", "url": "GANTI_WAHA_URL/api/sendText",
        "sendBody": true, "specifyBody": "json",
        "jsonBody": "={{ JSON.stringify({ session: 'default', chatId: $json.tujuan + '@c.us', text: $json.pesan }) }}"
      } },

    { "name": "Lapor Sukses", "type": "n8n-nodes-base.httpRequest", "typeVersion": 4.2,
      "position": [1100, -80],
      "parameters": {
        "method": "POST", "url": "https://madrasah-alfath-malang.web.id/api/v1/notifikasi/status",
        "sendHeaders": true,
        "headerParameters": { "parameters": [ { "name": "X-Bot-Secret", "value": "GANTI_SECRET" } ] },
        "sendBody": true, "specifyBody": "json",
        "jsonBody": "={{ JSON.stringify({ id: $('Loop Over Items').item.json.id, sukses: true }) }}"
      } },

    { "name": "Lapor Gagal", "type": "n8n-nodes-base.httpRequest", "typeVersion": 4.2,
      "position": [1100, 100],
      "parameters": {
        "method": "POST", "url": "https://madrasah-alfath-malang.web.id/api/v1/notifikasi/status",
        "sendHeaders": true,
        "headerParameters": { "parameters": [ { "name": "X-Bot-Secret", "value": "GANTI_SECRET" } ] },
        "sendBody": true, "specifyBody": "json",
        "jsonBody": "={{ JSON.stringify({ id: $('Loop Over Items').item.json.id, sukses: false, catatan: String($json.error || 'gagal kirim').slice(0,200) }) }}"
      } },

    { "name": "Jeda 8 Detik", "type": "n8n-nodes-base.wait", "typeVersion": 1.1,
      "position": [1320, 0], "webhookId": "jeda-notif",
      "parameters": { "amount": 8, "unit": "seconds" } }
  ],
  "connections": {
    "Setiap 5 Menit":   { "main": [ [ { "node": "Ambil Antrean", "type": "main", "index": 0 } ] ] },
    "Ambil Antrean":    { "main": [ [ { "node": "Pecah Items", "type": "main", "index": 0 } ] ] },
    "Pecah Items":      { "main": [ [ { "node": "Loop Over Items", "type": "main", "index": 0 } ] ] },
    "Loop Over Items":  { "main": [ [], [ { "node": "Kirim WAHA", "type": "main", "index": 0 } ] ] },
    "Kirim WAHA":       { "main": [ [ { "node": "Lapor Sukses", "type": "main", "index": 0 } ],
                                    [ { "node": "Lapor Gagal",  "type": "main", "index": 0 } ] ] },
    "Lapor Sukses":     { "main": [ [ { "node": "Jeda 8 Detik", "type": "main", "index": 0 } ] ] },
    "Lapor Gagal":      { "main": [ [ { "node": "Jeda 8 Detik", "type": "main", "index": 0 } ] ] },
    "Jeda 8 Detik":     { "main": [ [ { "node": "Loop Over Items", "type": "main", "index": 0 } ] ] }
  }
}
```

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
