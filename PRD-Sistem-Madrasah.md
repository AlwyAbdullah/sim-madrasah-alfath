# PRD — Sistem Informasi Madrasah

| | |
|---|---|
| **Nama Produk** | Sistem Informasi Madrasah (SIM-Madrasah) |
| **Versi Dokumen** | 1.0 |
| **Tanggal** | 15 Juni 2026 |
| **Pemilik Produk** | Alwy |
| **Status** | Draft |
| **Stack Teknologi** | Frontend: Next.js · Backend: Go (Golang) · Database: MySQL |

---

## 1. Ringkasan Eksekutif

SIM-Madrasah adalah aplikasi web internal untuk pengelolaan data akademik santri di sebuah madrasah. Fokus rilis pertama (MVP) adalah tiga fungsi inti:

1. **Autentikasi** — login berbasis username & password (password disimpan dalam bentuk hash).
2. **Dashboard/Overview** — ringkasan kehadiran dan nilai santri, dengan kemampuan memilih satu santri untuk melihat detailnya.
3. **Absensi Harian** — pencatatan kehadiran santri per kelas, mendukung input massal (batch) dan per-individu (Hadir/Izin/Sakit/Alpha) dengan kolom keterangan opsional.
4. **Penilaian** — input nilai santri terdiri dari tiga komponen: **Tugas, UTS, UAS**.

Dokumen ini mendefinisikan kebutuhan fungsional, non-fungsional, arsitektur, model data, dan rancangan API untuk mencapai tujuan tersebut.

---

## 2. Tujuan & Sasaran

### 2.1 Tujuan Bisnis
- Menggantikan pencatatan kehadiran & nilai manual (kertas/spreadsheet) dengan sistem terpusat.
- Mempercepat rekap kehadiran dan nilai santri.
- Menyediakan satu sumber data akademik yang konsisten dan dapat diaudit.

### 2.2 Sasaran Terukur (Success Metrics)
| Metrik | Target |
|---|---|
| Waktu mencatat absensi satu kelas | < 1 menit (dengan fitur "semua hadir") |
| Waktu input nilai satu kelas (1 komponen) | < 5 menit |
| Akurasi rekap kehadiran | 100% (tanpa hitung manual) |
| Ketersediaan sistem (uptime) | ≥ 99% jam operasional madrasah |

### 2.3 Di Luar Lingkup (Out of Scope) — MVP
- Pembayaran SPP / keuangan.
- Aplikasi mobile native (cukup web responsif).
- Portal wali santri / notifikasi orang tua.
- Modul jadwal pelajaran & kurikulum.
- Cetak rapor PDF resmi (dipertimbangkan untuk fase berikutnya).
- Registrasi mandiri (akun dibuat oleh admin).

---

## 3. Persona & Peran Pengguna

| Peran | Deskripsi | Akses Utama |
|---|---|---|
| **Admin** | Mengelola data master (santri, kelas, mata pelajaran, user). | Semua modul + manajemen user |
| **Guru/Ustadz** | Mencatat absensi & menginput nilai untuk kelas yang diampu. | Dashboard, Absensi, Nilai |
| **Kepala Madrasah (opsional)** | Melihat ringkasan/laporan, hanya baca. | Dashboard (read-only) |

> **Catatan MVP:** Untuk rilis pertama, minimal terdapat peran **Admin** dan **Guru**. Peran Kepala Madrasah bisa dimodelkan sebagai akun read-only.

---

## 4. Kebutuhan Fungsional

### FR-1 — Halaman Login (Autentikasi)

**Deskripsi:** Saat membuka aplikasi, pengguna yang belum login diarahkan ke halaman login. Login menggunakan **username + password**.

**Detail:**
- FR-1.1 Form login berisi field: `username`, `password`, tombol **Masuk**.
- FR-1.2 Password **tidak pernah** disimpan plaintext. Gunakan hashing **bcrypt** (cost ≥ 12) atau **argon2id**.
- FR-1.3 Validasi kredensial dilakukan di backend Go. Bila salah, tampilkan pesan generik ("Username atau password salah") tanpa membocorkan field mana yang salah.
- FR-1.4 Setelah login berhasil, server menerbitkan **JWT** (access token) + opsional refresh token; disimpan sebagai **HttpOnly cookie** (lebih aman daripada localStorage).
- FR-1.5 Sesi memiliki masa berlaku (mis. access token 60 menit). Token kedaluwarsa → diarahkan kembali ke login.
- FR-1.6 Tombol **Logout** menghapus cookie/sesi.
- FR-1.7 Rate limiting pada endpoint login (mis. maks 5 percobaan gagal / 15 menit per IP/username) untuk mencegah brute force.
- FR-1.8 Semua halaman selain login dilindungi (protected route) — akses tanpa sesi valid ditolak.

**Kriteria Penerimaan:**
- Pengguna dengan kredensial benar masuk ke dashboard.
- Pengguna dengan kredensial salah tetap di halaman login dengan pesan error.
- Mengakses URL dashboard langsung tanpa login → diarahkan ke `/login`.
- Di database, kolom password berisi hash (bukan teks asli).

---

### FR-2 — Dashboard / Overview

**Deskripsi:** Setelah login, pengguna langsung melihat halaman ringkasan kehadiran & nilai santri. Santri dapat dipilih, dan sistem otomatis menampilkan detail santri tersebut.

**Detail:**
- FR-2.1 **Kartu ringkasan (KPI)** di bagian atas:
  - Total santri.
  - Persentase kehadiran hari ini (atau periode terpilih).
  - Jumlah Hadir / Izin / Sakit / Alpha hari ini.
  - Rata-rata nilai (opsional, per kelas/periode).
- FR-2.2 **Filter** periode/kelas (mis. pilih kelas, rentang tanggal).
- FR-2.3 **Daftar/pemilih santri** — dropdown pencarian atau tabel yang bisa diklik. Saat satu santri dipilih, panel **Detail Santri** terisi otomatis tanpa reload penuh.
- FR-2.4 **Detail Santri** menampilkan:
  - Identitas: nama, NIS, kelas, jenis kelamin.
  - Rekap kehadiran: jumlah Hadir/Izin/Sakit/Alpha + persentase, dengan riwayat absensi terbaru.
  - Rekap nilai per mata pelajaran: Tugas, UTS, UAS, dan **Nilai Akhir** (perhitungan terbobot, lihat FR-4.5).
- FR-2.5 Tampilan responsif (desktop & tablet minimal).

**Kriteria Penerimaan:**
- Memilih santri langsung memperbarui panel detail (kehadiran + nilai) tanpa pindah halaman.
- KPI berubah sesuai filter kelas/periode.

---

### FR-3 — Absensi Harian

**Deskripsi:** Mencatat kehadiran santri per kelas per tanggal. Mendukung input massal & per-individu.

**Status Kehadiran:** `Hadir` · `Izin` · `Sakit` · `Alpha`.

**Detail:**
- FR-3.1 Pengguna memilih **kelas** dan **tanggal** (default: hari ini).
- FR-3.2 Sistem menampilkan daftar santri kelas tersebut dalam bentuk tabel.
- FR-3.3 **Aksi massal "Tandai Semua Hadir"** — satu checkbox/tombol di header yang otomatis menandai seluruh santri sebagai **Hadir**.
- FR-3.4 Setelah massal Hadir, pengguna tetap dapat **mengubah** status santri tertentu menjadi Izin/Sakit/Alpha (override per baris).
- FR-3.5 Setiap baris memiliki **kolom keterangan opsional** (free text), mis. alasan izin. Boleh dikosongkan.
- FR-3.6 Mode penyimpanan **upsert**: jika absensi untuk (santri, tanggal) sudah ada, disimpan = update; jika belum, insert. Mencegah duplikasi.
- FR-3.7 Tombol **Simpan** menyimpan seluruh kelas dalam satu permintaan (batch). Tampilkan konfirmasi sukses.
- FR-3.8 Indikator visual jumlah Hadir/Izin/Sakit/Alpha yang terhitung saat pengisian.
- FR-3.9 (Opsional) Mencegah/peringatan saat menyimpan absensi untuk tanggal yang sudah lampau di luar batas tertentu.

**Aturan Bisnis:**
- Satu santri hanya boleh punya **satu** catatan absensi per tanggal (unique constraint `santri_id + tanggal`).
- Default status saat form dibuka: belum ditandai (atau Hadir, sesuai preferensi — direkomendasikan kosong agar disengaja).

**Kriteria Penerimaan:**
- Klik "Semua Hadir" mencentang semua santri sebagai Hadir.
- Bisa mengubah sebagian santri menjadi Izin/Sakit/Alpha setelah massal.
- Keterangan boleh kosong dan tetap tersimpan.
- Menyimpan ulang tanggal yang sama memperbarui data, bukan menggandakan.

---

### FR-4 — Penilaian (Input Nilai Santri)

**Deskripsi:** Menginput nilai santri per mata pelajaran per periode, terdiri dari tiga komponen: **Tugas, UTS, UAS**.

**Detail:**
- FR-4.1 Pengguna memilih **kelas**, **mata pelajaran**, dan **periode** (mis. semester/tahun ajaran).
- FR-4.2 Tabel berisi daftar santri dengan tiga kolom input numerik: **Tugas**, **UTS**, **UAS** (rentang 0–100).
- FR-4.3 Validasi: nilai 0–100, boleh desimal (mis. satu angka di belakang koma), boleh kosong (belum dinilai).
- FR-4.4 Penyimpanan batch (satu permintaan untuk satu kelas + mapel + periode), bersifat **upsert**.
- FR-4.5 **Nilai Akhir** dihitung otomatis dengan bobot konfigurabel. Default mengikuti standar yang sudah dipakai: **Tugas 30% · UTS 30% · UAS 40%**.
  - `Nilai Akhir = (Tugas×0.30) + (UTS×0.30) + (UAS×0.40)`
  - Bobot disimpan sebagai konfigurasi agar mudah diubah.
- FR-4.6 Kolom Nilai Akhir bersifat *read-only* (terhitung), ditampilkan langsung saat pengguna mengisi.
- FR-4.7 Tombol **Simpan** menyimpan seluruh nilai kelas.

**Aturan Bisnis:**
- Satu kombinasi unik: `santri_id + mata_pelajaran_id + periode` (unique constraint).

**Kriteria Penerimaan:**
- Mengisi Tugas/UTS/UAS otomatis menampilkan Nilai Akhir sesuai bobot 30/30/40.
- Input di luar 0–100 ditolak dengan pesan validasi.
- Nilai tersimpan dan muncul kembali saat membuka kelas/mapel/periode yang sama.

---

### FR-5 — Manajemen Data Master (Pendukung)

Diperlukan agar FR-1 s/d FR-4 berfungsi.

- FR-5.1 **User**: admin dapat membuat/menonaktifkan akun guru (username, nama, peran, password awal).
- FR-5.2 **Santri**: tambah/edit/nonaktif (NIS, nama, jenis kelamin, kelas).
- FR-5.3 **Kelas**: tambah/edit (nama kelas, tingkat, tahun ajaran).
- FR-5.4 **Mata Pelajaran**: tambah/edit.
- FR-5.5 **Periode/Tahun Ajaran & Semester**: penanda periode penilaian.

> Pada MVP, manajemen master boleh berupa CRUD sederhana atau bahkan seed data awal, sepanjang FR-1–FR-4 dapat dijalankan.

---

## 5. Kebutuhan Non-Fungsional

| Kategori | Kebutuhan |
|---|---|
| **Keamanan** | Password di-hash (bcrypt/argon2id). JWT via HttpOnly+Secure cookie. HTTPS wajib di produksi. Proteksi terhadap SQL injection (prepared statement), XSS, CSRF. Rate limiting login. |
| **Performa** | Respons API < 300 ms untuk operasi umum pada beban normal. Halaman dashboard interaktif < 2 dtk. |
| **Skalabilitas** | Mendukung ratusan santri & puluhan kelas tanpa penurunan performa berarti. |
| **Keandalan** | Operasi simpan bersifat transaksional (atomic). Tidak ada data ganda (unique constraint). |
| **Usability** | Antarmuka Bahasa Indonesia, sederhana, minim klik. Fitur "semua hadir" mempercepat input. |
| **Audit** | Setiap perubahan absensi/nilai mencatat `created_by`, `updated_by`, timestamp. |
| **Kompatibilitas** | Browser modern (Chrome, Edge, Firefox) versi terkini; responsif desktop & tablet. |
| **Maintainability** | Kode terstruktur (layered), konfigurasi via environment variable, migrasi DB terversion. |
| **Lokalitas** | Zona waktu server WIB (Asia/Jakarta) untuk tanggal absensi. |

---

## 6. Arsitektur Sistem

### 6.1 Gambaran Umum

```
┌──────────────────┐      HTTPS / JSON       ┌──────────────────┐         ┌────────────┐
│   Next.js (FE)   │  ───────────────────▶   │   Go API (BE)    │  ────▶  │   MySQL    │
│  - UI / Pages    │   JWT (HttpOnly cookie) │  - REST handlers │  SQL    │  - Tabel   │
│  - App Router    │  ◀───────────────────   │  - Auth/Middleware│ ◀────  │  - Index   │
│  - Fetch/SWR     │                         │  - Service/Repo  │         └────────────┘
└──────────────────┘                         └──────────────────┘
```

### 6.2 Frontend — Next.js
- App Router, komponen server & client sesuai kebutuhan.
- Proteksi route via middleware (cek sesi/cookie).
- State data via React Query/SWR atau fetch bawaan.
- Komponen UI: tabel absensi, form nilai, kartu dashboard, pemilih santri.

### 6.3 Backend — Go (Golang)
- Struktur berlapis: `handler (HTTP) → service (logika bisnis) → repository (akses DB)`.
- Router: `chi`/`gin`/`echo` (pilih satu; rekomendasi `chi` atau `echo`).
- Middleware: autentikasi JWT, logging, recovery, CORS, rate limit.
- Hash password: `golang.org/x/crypto/bcrypt`.
- JWT: `github.com/golang-jwt/jwt`.
- Validasi input di service.
- Migrasi DB: `golang-migrate` atau `goose`.

### 6.4 Database — MySQL
- InnoDB, transaksi, foreign key.
- Index pada kolom pencarian/filter (kelas, tanggal, periode).
- Unique constraint untuk mencegah duplikasi absensi/nilai.

---

## 7. Model Data (Skema MySQL)

> Tipe & nama dapat disesuaikan saat implementasi. Semua tabel memiliki `created_at`, `updated_at`.

```sql
-- Pengguna sistem (admin/guru)
CREATE TABLE users (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(50)  NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,        -- bcrypt/argon2id
  nama          VARCHAR(120) NOT NULL,
  role          ENUM('admin','guru','kepala') NOT NULL DEFAULT 'guru',
  is_active     TINYINT(1)   NOT NULL DEFAULT 1,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Tahun ajaran / periode
CREATE TABLE periode (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  nama         VARCHAR(50) NOT NULL,           -- mis. "2025/2026 - Ganjil"
  tahun_ajaran VARCHAR(20) NOT NULL,
  semester     ENUM('ganjil','genap') NOT NULL,
  is_active    TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY uq_periode (tahun_ajaran, semester)
);

-- Kelas
CREATE TABLE kelas (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  nama       VARCHAR(50) NOT NULL,             -- mis. "7A"
  tingkat    VARCHAR(20),
  wali_id    BIGINT NULL,                      -- FK users (guru)
  FOREIGN KEY (wali_id) REFERENCES users(id)
);

-- Santri
CREATE TABLE santri (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  nis           VARCHAR(30) UNIQUE,
  nama          VARCHAR(120) NOT NULL,
  jenis_kelamin ENUM('L','P') NOT NULL,
  kelas_id      BIGINT NOT NULL,
  is_active     TINYINT(1) NOT NULL DEFAULT 1,
  FOREIGN KEY (kelas_id) REFERENCES kelas(id)
);

-- Mata pelajaran
CREATE TABLE mata_pelajaran (
  id   BIGINT PRIMARY KEY AUTO_INCREMENT,
  kode VARCHAR(20) UNIQUE,
  nama VARCHAR(120) NOT NULL
);

-- Absensi harian
CREATE TABLE absensi (
  id          BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id   BIGINT NOT NULL,
  tanggal     DATE   NOT NULL,
  status      ENUM('hadir','izin','sakit','alpha') NOT NULL,
  keterangan  VARCHAR(255) NULL,               -- opsional
  created_by  BIGINT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_absensi (santri_id, tanggal),  -- cegah duplikat per hari
  FOREIGN KEY (santri_id) REFERENCES santri(id),
  FOREIGN KEY (created_by) REFERENCES users(id),
  INDEX idx_absensi_tanggal (tanggal)
);

-- Nilai
CREATE TABLE nilai (
  id                BIGINT PRIMARY KEY AUTO_INCREMENT,
  santri_id         BIGINT NOT NULL,
  mata_pelajaran_id BIGINT NOT NULL,
  periode_id        BIGINT NOT NULL,
  tugas             DECIMAL(5,2) NULL,         -- 0-100
  uts               DECIMAL(5,2) NULL,
  uas               DECIMAL(5,2) NULL,
  -- nilai_akhir dihitung di aplikasi (Tugas 30% + UTS 30% + UAS 40%)
  nilai_akhir       DECIMAL(5,2) NULL,
  created_by        BIGINT NULL,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_nilai (santri_id, mata_pelajaran_id, periode_id),
  FOREIGN KEY (santri_id) REFERENCES santri(id),
  FOREIGN KEY (mata_pelajaran_id) REFERENCES mata_pelajaran(id),
  FOREIGN KEY (periode_id) REFERENCES periode(id)
);
```

**Relasi:** `kelas 1–N santri` · `santri 1–N absensi` · `santri 1–N nilai` · `mata_pelajaran 1–N nilai` · `periode 1–N nilai`.

---

## 8. Rancangan API (REST)

Base URL: `/api/v1`. Format: JSON. Auth: JWT via HttpOnly cookie (kecuali login).

### 8.1 Autentikasi
| Method | Endpoint | Deskripsi |
|---|---|---|
| POST | `/auth/login` | Body `{username, password}` → set cookie + `{user}` |
| POST | `/auth/logout` | Hapus sesi |
| GET | `/auth/me` | Info user dari sesi aktif |

### 8.2 Dashboard
| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/dashboard/summary?kelas_id=&tanggal=&periode_id=` | KPI ringkasan |
| GET | `/santri?kelas_id=&q=` | Daftar/cari santri |
| GET | `/santri/{id}/detail?periode_id=` | Detail santri: rekap kehadiran + nilai |

### 8.3 Absensi
| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/absensi?kelas_id=&tanggal=` | Daftar santri + status absensi tanggal tsb |
| POST | `/absensi/batch` | Simpan/upsert absensi satu kelas |

Contoh body `POST /absensi/batch`:
```json
{
  "kelas_id": 3,
  "tanggal": "2026-06-15",
  "items": [
    { "santri_id": 10, "status": "hadir" },
    { "santri_id": 11, "status": "izin", "keterangan": "acara keluarga" },
    { "santri_id": 12, "status": "sakit" }
  ]
}
```

### 8.4 Nilai
| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/nilai?kelas_id=&mata_pelajaran_id=&periode_id=` | Daftar nilai untuk diisi/edit |
| POST | `/nilai/batch` | Simpan/upsert nilai satu kelas+mapel+periode |

Contoh body `POST /nilai/batch`:
```json
{
  "kelas_id": 3,
  "mata_pelajaran_id": 5,
  "periode_id": 1,
  "items": [
    { "santri_id": 10, "tugas": 80, "uts": 75, "uas": 90 },
    { "santri_id": 11, "tugas": 70, "uts": 65, "uas": 80 }
  ]
}
```
> `nilai_akhir` dihitung server: `tugas*0.3 + uts*0.3 + uas*0.4`.

### 8.5 Master (Admin)
| Method | Endpoint |
|---|---|
| CRUD | `/users`, `/santri`, `/kelas`, `/mata-pelajaran`, `/periode` |

**Format error standar:**
```json
{ "error": { "code": "INVALID_CREDENTIALS", "message": "Username atau password salah" } }
```

---

## 9. Alur Pengguna Utama (User Flows)

**Alur 1 — Login → Dashboard**
1. Buka aplikasi → diarahkan ke `/login`.
2. Isi username & password → submit.
3. Sukses → redirect ke `/dashboard`, KPI & daftar santri tampil.

**Alur 2 — Lihat detail santri**
1. Di dashboard, ketik/pilih nama santri.
2. Panel detail otomatis menampilkan rekap kehadiran & nilai.

**Alur 3 — Absensi harian (batch)**
1. Buka menu Absensi → pilih kelas (tanggal = hari ini).
2. Klik "Tandai Semua Hadir".
3. Ubah beberapa santri jadi Izin/Sakit/Alpha; isi keterangan bila perlu.
4. Klik Simpan → konfirmasi sukses.

**Alur 4 — Input nilai**
1. Buka menu Nilai → pilih kelas, mata pelajaran, periode.
2. Isi Tugas/UTS/UAS; Nilai Akhir muncul otomatis.
3. Klik Simpan.

---

## 10. Rancangan Antarmuka (Ringkas)

- **Login:** kartu di tengah, logo madrasah, 2 field + tombol.
- **Layout utama:** sidebar (Dashboard, Absensi, Nilai, Master) + topbar (nama user, logout).
- **Dashboard:** baris kartu KPI → pemilih santri → panel detail (tab Kehadiran / Nilai).
- **Absensi:** header (kelas, tanggal, tombol "Semua Hadir", ringkasan jumlah) → tabel (No, Nama, status radio/select, keterangan) → tombol Simpan sticky.
- **Nilai:** header (kelas, mapel, periode) → tabel (Nama, Tugas, UTS, UAS, Nilai Akhir read-only) → tombol Simpan.

---

## 11. Rencana Rilis & Prioritas

| Fase | Lingkup | Prioritas |
|---|---|---|
| **MVP (v1.0)** | FR-1 Login, FR-2 Dashboard, FR-3 Absensi, FR-4 Nilai, FR-5 master minimal | Must-have |
| **v1.1** | Laporan/rekap ekspor (Excel/PDF), filter lanjutan, audit log UI | Should-have |
| **v1.2** | Multi-peran lengkap, portal wali, notifikasi | Could-have |

**Saran milestone implementasi MVP:**
1. Setup proyek (Next.js, Go, MySQL, migrasi) + skema DB + seed.
2. Auth (login, JWT, middleware, hash).
3. Master data minimal (santri, kelas, mapel, periode).
4. Absensi (GET + batch upsert + UI "semua hadir").
5. Nilai (GET + batch upsert + perhitungan akhir).
6. Dashboard (summary + detail santri).
7. Pengujian, hardening keamanan, deploy.

---

## 12. Risiko & Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Kebocoran password | Tinggi | Hash bcrypt/argon2id, HTTPS, rate limit, tidak log password |
| Duplikasi data absensi/nilai | Sedang | Unique constraint + upsert |
| Input nilai salah rentang | Rendah | Validasi 0–100 di FE & BE |
| Zona waktu salah (tanggal absen) | Sedang | Standarkan WIB di server & query |
| Kehilangan data | Tinggi | Backup berkala MySQL |

---

## 13. Pertanyaan Terbuka

1. Apakah bobot nilai **30/30/40** bersifat tetap atau perlu bisa diubah per mata pelajaran? (PRD mengasumsikan default 30/30/40, konfigurabel.)
2. Apakah guru hanya bisa mengisi absensi/nilai untuk kelas yang diampu, atau semua kelas? (PRD mengasumsikan pembatasan per pengampu untuk v1.1; MVP boleh akses semua.)
3. Apakah perlu cetak rapor/rekap resmi pada MVP? (Diasumsikan tidak — masuk v1.1.)
4. Berapa jumlah santri & kelas perkiraan (untuk kapasitas)?

---

*Dokumen ini adalah baseline. Perbarui seiring keputusan teknis & masukan stakeholder.*
