# SIM-Madrasah

Sistem Informasi Madrasah — manajemen kehadiran & nilai santri.

**Stack:** Next.js (frontend) · Go/Golang (backend) · MySQL.
PRD lengkap: [PRD-Sistem-Madrasah.md](PRD-Sistem-Madrasah.md).

## Fitur (MVP)
- Login username + password (password di-hash bcrypt, sesi JWT via cookie HttpOnly).
- Dashboard ringkasan kehadiran & nilai; pilih santri → detail otomatis.
- Absensi harian batch ("Tandai Semua Hadir") + status Hadir/Izin/Sakit/Alpha + keterangan opsional.
- Input nilai Tugas/UTS/UAS dengan **Nilai Akhir = Tugas 30% + UTS 30% + UAS 40%**.
- **Ekspor nilai ke Excel (.xlsx)** per kelas + mata pelajaran + periode.
- **Ekspor rekap absensi bulanan ke Excel** (matriks per tanggal + total H/I/S/A per santri).
- **Manajemen master (khusus admin):** CRUD Santri, Kelas, Mata Pelajaran, Periode, dan User.
- Guru **tidak dibatasi** pada kelas tertentu — semua kelas dapat diakses.
- Nama kelas bebas karakter (mis. `4A`, `4B`) dengan constraint unik anti-duplikat.

## Struktur
```
backend/    API Go (chi router, MySQL, JWT)
  cmd/server/main.go      entry point
  internal/               config, db, auth, middleware, handlers, models
  migrations/             001_init.sql, 002_seed.sql
frontend/   Next.js (App Router, TypeScript)
  app/                    login, dashboard, absensi, nilai
  lib/api.ts              API client
```

## Menjalankan

### 1. Database (MySQL)
```bash
mysql -u root -p < backend/migrations/001_init.sql
mysql -u root -p < backend/migrations/002_seed.sql
```

### 2. Backend (Go)
```bash
cd backend
cp .env.example .env      # sesuaikan kredensial MySQL & JWT_SECRET
go run ./cmd/server
# berjalan di http://localhost:8080
```

### 3. Frontend (Next.js)
```bash
cd frontend
cp .env.local.example .env.local
npm install
npm run dev
# buka http://localhost:3000
```

## Akun default
| Username | Password | Role |
|---|---|---|
| admin | admin123 | admin |
| guru | admin123 | guru |

> Ganti password & `JWT_SECRET` sebelum produksi. Aktifkan `COOKIE_SECURE=true` di HTTPS.

## Endpoint utama (`/api/v1`)
| Method | Path | Keterangan |
|---|---|---|
| POST | `/auth/login` | login (rate-limited) |
| POST | `/auth/logout` | logout |
| GET | `/auth/me` | info user |
| GET | `/dashboard/summary` | KPI kehadiran |
| GET | `/santri/{id}/detail` | detail santri |
| GET/POST | `/absensi`, `/absensi/batch` | ambil/simpan absensi |
| GET | `/absensi/export?kelas_id=&bulan=YYYY-MM` | ekspor rekap absensi bulanan |
| GET/POST | `/nilai`, `/nilai/batch` | ambil/simpan nilai |
| GET | `/nilai/export` | ekspor nilai ke Excel |
| GET | `/kelas`, `/mata-pelajaran`, `/periode`, `/santri` | master data (baca, semua role) |

### Master CRUD — khusus admin
| Method | Path |
|---|---|
| POST/PUT/DELETE | `/kelas`, `/kelas/{id}` |
| POST/PUT/DELETE | `/santri`, `/santri/{id}` (delete = nonaktif) |
| POST/PUT/DELETE | `/mata-pelajaran`, `/mata-pelajaran/{id}` |
| POST/PUT/DELETE | `/periode`, `/periode/{id}` |
| GET/POST/PUT/DELETE | `/users`, `/users/{id}` (delete = nonaktif) |
