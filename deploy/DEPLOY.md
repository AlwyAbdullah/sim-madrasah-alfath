> ## ✅ PRODUKSI AKTIF
> - **URL:** https://madrasah-alfath-malang.web.id (www juga aktif). IP lama `103.175.219.47` otomatis redirect ke domain.
> - **Server:** Ubuntu 22.04, user `deploy`, kode di `/home/deploy/sim-madrasah`.
> - **Backend** Go `:8090` → service `sim-madrasah-backend`. **Frontend** Next `:3000` → `sim-madrasah-frontend`. DB MySQL `sim_madrasah` (user `sim_user`).
> - **Nginx** `sites-available/sim-madrasah` (server_name = domain + www + IP). Tidak mengganggu situs `sayalulus` yang ada.
> - **Update rilis berikutnya:** jalankan `bash deploy/redeploy.sh` dari mesin lokal (Git Bash).
> - **Log:** `journalctl -u sim-madrasah-backend -e` / `... -frontend -e`.
> - **HTTPS:** ✅ aktif (Let's Encrypt, auto-renew via `certbot.timer`; kedaluwarsa 26 Okt 2026). `COOKIE_SECURE=true` & `CORS_ORIGIN=https://madrasah-alfath-malang.web.id` di `.env`.
>   Perpanjangan otomatis; cek manual: `sudo certbot renew --dry-run`.
> - **WAHA (WhatsApp bot notifikasi ortu):** terpasang via Docker di VPS ini sendiri, `devlikeapro/waha:noweb`
>   (compose di `/home/deploy/waha/`, kredensial di `/home/deploy/waha/.env`, perm 600).
>   Bind **hanya** ke `127.0.0.1:3001` (tidak publik) — backend memanggil langsung via worker bawaan
>   (`internal/notifworker`). RAM dibatasi 400 MB. Sesi WhatsApp sudah ter-pairing: `default`.
>   ⏸️ **Saat ini DIMATIKAN** (container `docker compose stop` + saklar nonaktif dari halaman
>   **Notifikasi WA**) — sesi tetap tersimpan, tidak perlu scan ulang. Nyalakan lagi:
>   `cd /home/deploy/waha && docker compose start` lalu toggle "Aktifkan" di halaman **Notifikasi WA**.
>   Detail & alternatif (n8n) di `docs/BOT-NOTIFIKASI-WA.md`.
> - **Backup database:** ✅ otomatis **tiap hari 02:00** via `sim-madrasah-backup.timer`
>   → `/home/deploy/backups/`. Retensi: **30 hari** harian + backup tanggal 1 disimpan **1 tahun**.
>   Skrip memverifikasi hasilnya (uji gzip, jumlah tabel, penanda "Dump completed") dan gagal
>   bila mencurigakan. Ikut menyimpan salinan konfigurasi (`config-*.tar.gz`, perm 600, berisi rahasia).
>   Cek: `systemctl list-timers sim-madrasah-backup.timer` · `journalctl -u sim-madrasah-backup -e`
>   Jalankan manual: `sudo systemctl start sim-madrasah-backup.service`
>   **Cara pulih → lihat bagian "Pemulihan Darurat" di bawah.**
>
> Bagian di bawah = panduan generik/dari nol (referensi).

---

## 🚨 Pemulihan Darurat (restore database)

Backup harian ada di `/home/deploy/backups/` sebagai `sim_madrasah-TANGGAL_JAM.sql.gz`.

**1. Lihat backup yang tersedia**
```bash
ls -lht /home/deploy/backups/
```

**2. Uji dulu ke database sementara (SANGAT disarankan sebelum menimpa yang asli)**
```bash
FILE=/home/deploy/backups/sim_madrasah-2026-08-07_155640.sql.gz   # ganti sesuai pilihan
sudo mysql -e "DROP DATABASE IF EXISTS uji_pulih; CREATE DATABASE uji_pulih CHARACTER SET utf8mb4;"
zcat "$FILE" | sudo mysql uji_pulih
sudo mysql -e "SELECT COUNT(*) AS santri FROM uji_pulih.santri; SELECT COUNT(*) AS spp FROM uji_pulih.spp;"
```
Bila angkanya masuk akal, lanjut. Bersihkan setelahnya: `sudo mysql -e "DROP DATABASE uji_pulih;"`

**3. Pulihkan ke database asli** (⚠️ menimpa data sekarang — pastikan langkah 2 sudah dicek)
```bash
sudo systemctl stop sim-madrasah-backend           # hentikan dulu agar tidak ada tulisan masuk
sudo mysqldump --single-transaction sim_madrasah | gzip > /home/deploy/backups/SEBELUM-PULIH-$(date +%F_%H%M%S).sql.gz
zcat "$FILE" | sudo mysql sim_madrasah
sudo systemctl start sim-madrasah-backend
```
Langkah `mysqldump` di tengah membuat cadangan kondisi **sebelum** dipulihkan — jaring pengaman
kalau ternyata salah pilih berkas.

**4. Bila server benar-benar hilang (bangun dari nol)**
Selain `.sql.gz`, ada `config-*.tar.gz` berisi `.env` backend/frontend, konfigurasi WAHA,
nginx, dan unit systemd. Bongkar dengan `sudo tar xzf config-XXX.tar.gz -C /` setelah aplikasi
dipasang ulang, lalu pulihkan database seperti langkah 3.

> 💡 Backup tersimpan **di VPS yang sama**. Untuk perlindungan terhadap kegagalan total server,
> unduh berkalanya ke komputer lain:
> `scp -i ~/.ssh/id_ed25519 deploy@103.175.219.47:/home/deploy/backups/sim_madrasah-*.sql.gz .`

# Panduan Deploy SIM-Madrasah ke VPS

Arsitektur produksi (1 domain, same-origin):

```
Internet → Nginx (443/80) ─┬─ /api/*  → Go backend  (127.0.0.1:8080, systemd)
                           └─ /*      → Next.js     (127.0.0.1:3000, systemd)
                                         └─ MySQL/MariaDB (127.0.0.1:3306)
```

Karena satu domain, tidak ada masalah CORS dan cookie JWT bersifat first-party.

---

## 0. Prasyarat di VPS
- Linux (Ubuntu/Debian diasumsikan), akses sudo.
- Terinstal: **Go ≥ 1.22**, **Node ≥ 20**, **MySQL/MariaDB**, **Nginx**, **git**, **certbot**.
```bash
sudo apt update
sudo apt install -y nginx git mariadb-server certbot python3-certbot-nginx
# Go & Node: pasang via paket resmi / nvm bila belum ada
```

## 1. Ambil kode
```bash
sudo mkdir -p /opt/sim-madrasah
sudo chown -R $USER:$USER /opt/sim-madrasah
# salin folder backend/ dan frontend/ ke /opt/sim-madrasah (git clone / scp / rsync)
```

## 2. Database
```bash
sudo mysql <<'SQL'
CREATE DATABASE IF NOT EXISTS sim_madrasah CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'sim_user'@'localhost' IDENTIFIED BY 'PASSWORD_KUAT';
GRANT ALL PRIVILEGES ON sim_madrasah.* TO 'sim_user'@'localhost';
FLUSH PRIVILEGES;
SQL

cd /opt/sim-madrasah/backend
mysql -u sim_user -p sim_madrasah < migrations/001_init.sql
mysql -u sim_user -p sim_madrasah < migrations/003_seed_alfath.sql   # data santri nyata
mysql -u sim_user -p sim_madrasah < migrations/004_spp.sql
# JANGAN jalankan 002_seed.sql (itu data dummy)
```

## 3. Backend (Go)
```bash
cd /opt/sim-madrasah/backend
cp .env.production.example .env
nano .env        # isi DB_PASSWORD, JWT_SECRET (openssl rand -hex 32), CORS_ORIGIN=domain, COOKIE_SECURE=true
go build -o sim-server ./cmd/server
```

## 4. Frontend (Next.js)
```bash
cd /opt/sim-madrasah/frontend
cp .env.production.example .env.production    # NEXT_PUBLIC_API_BASE=/api/v1
npm ci
npm run build        # variabel NEXT_PUBLIC_* dibaca di tahap ini
```

## 5. systemd (jalan otomatis & restart)
```bash
sudo cp /opt/sim-madrasah/deploy/sim-backend.service /etc/systemd/system/
sudo cp /opt/sim-madrasah/deploy/sim-frontend.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now sim-backend sim-frontend
systemctl status sim-backend sim-frontend --no-pager
```

## 6. Nginx + HTTPS
```bash
sudo cp /opt/sim-madrasah/deploy/nginx-sim-madrasah.conf /etc/nginx/sites-available/sim-madrasah
sudo nano /etc/nginx/sites-available/sim-madrasah   # ganti server_name
sudo ln -s /etc/nginx/sites-available/sim-madrasah /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d madrasah.example.com         # pasang sertifikat HTTPS
```

## 7. Setelah live (WAJIB)
- Login `admin` / `admin123` → **ganti password** lewat Master → User (atau buat admin baru, nonaktifkan default).
- Pastikan `COOKIE_SECURE=true` dan situs diakses via **https://**.
- Backup terjadwal: `mysqldump -u sim_user -p sim_madrasah > backup-$(date +%F).sql` (mis. via cron harian).

## Update versi berikutnya
```bash
cd /opt/sim-madrasah && git pull        # atau rsync ulang
cd backend && go build -o sim-server ./cmd/server && sudo systemctl restart sim-backend
cd ../frontend && npm ci && npm run build && sudo systemctl restart sim-frontend
```

## Troubleshooting
- `journalctl -u sim-backend -e` / `journalctl -u sim-frontend -e` — log service.
- 502 di Nginx → cek service backend/frontend hidup (`systemctl status`).
- Login gagal walau benar → cek `COOKIE_SECURE` (harus false bila belum HTTPS) & domain.
