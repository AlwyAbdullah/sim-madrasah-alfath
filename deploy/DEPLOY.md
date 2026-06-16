> ## ✅ PRODUKSI AKTIF
> - **URL:** http://103.175.219.47  · Login awal: `admin` / `admin123` (**WAJIB ganti**).
> - **Server:** Ubuntu 22.04, user `deploy`, kode di `/home/deploy/sim-madrasah`.
> - **Backend** Go `:8090` → service `sim-madrasah-backend`. **Frontend** Next `:3000` → `sim-madrasah-frontend`. DB MySQL `sim_madrasah` (user `sim_user`).
> - **Nginx** `sites-available/sim-madrasah` (server_name = IP). Tidak mengganggu situs `sayalulus` yang ada.
> - **Update rilis berikutnya:** jalankan `bash deploy/redeploy.sh` dari mesin lokal (Git Bash).
> - **Log:** `journalctl -u sim-madrasah-backend -e` / `... -frontend -e`.
> - **HTTPS (opsional):** arahkan hostname (mis. `sim.103-175-219-47.sslip.io`) → set `server_name` di nginx → `sudo certbot --nginx` → set `COOKIE_SECURE=true` di `.env`, restart backend.
>
> Bagian di bawah = panduan generik/dari nol (referensi).

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
