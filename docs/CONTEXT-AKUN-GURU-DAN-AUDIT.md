# Konteks: Akun per Guru, Superadmin, Wali Kelas & Pencatatan Aktivitas

Status: **rancangan revisi 3 — belum dikerjakan.**

---

## 1. Keputusan yang sudah diambil

| Hal | Keputusan |
|---|---|
| Gelar `U ` | singkatan "Ustadz" — dibuang dari username |
| Peran `kepala` | tidak dipakai |
| Superadmin | **3 orang**: Muhammad Al Masyhur, Alwy Alaydrus, Sholeh Assegaf |
| Akun guru | **20 akun** — seluruh guru di master (lihat 1.1 kenapa berubah dari 18) |
| Password awal | `guru123` untuk semua, tanpa kewajiban ganti |
| Ganti password | tersedia di halaman user |
| Lihat password | **tidak disimpan terbaca** — dipakai status password + tombol reset (4.4) |
| Wali kelas | ditetapkan, sebagian kelas punya **dua** wali; Kelas 2/4 Pagi dikosongkan dulu |

### 1.1 Satu bentrokan yang harus diputuskan

Sebelumnya Anda memilih **satu akun `admin` dipakai bersama**. Permintaan wali kelas yang
baru membuat itu tidak bisa jalan, dan alasannya konkret:

- **Muhammad Al Masyhur** wali **Kelas 4**
- **Alwy Alaydrus** wali **Kelas 5**

Keduanya superadmin. Kalau berbagi satu akun `admin`, maka Kelas 4 dan Kelas 5 menunjuk ke
**akun yang sama**, sehingga sistem tidak punya cara membedakan siapa wali yang mana.
Pengingat "Ustadz X, Kelas 4 belum diabsen" jadi mustahil — akan terkirim ke satu akun untuk
dua kelas berbeda, tanpa nama yang jelas. Fitur wali kelas praktis mati sejak hari pertama.

**Karena itu rancangan ini memakai akun terpisah untuk ketiga superadmin.** Tiap orang punya
identitas sendiri, wali kelas berfungsi, dan log jadi berarti. Akun `admin` yang lama
dinonaktifkan setelah akun baru dibagikan — **tidak dihapus**, karena 511 absensi + 496 nilai
+ 419 SPP menunjuk ke sana dan riwayat itu harus utuh sebagai jejak "era akun bersama".

Kalau Anda tetap ingin satu akun `admin` bersama, bisa — tapi konsekuensinya wali kelas untuk
Kelas 4 dan Kelas 5 tidak bisa dibedakan, dan tahap 4 (alert Telegram per orang) tidak berlaku
untuk Anda berdua. Beri tahu kalau memang itu yang dipilih.

---

## 2. Masalah yang diselesaikan

Kolom `created_by` sudah ada di hampir semua tabel penting, tapi **isinya tidak bermakna**:

```
SELECT DISTINCT created_by FROM absensi;   ->  1, 2
```

| Akun | Absensi | Nilai | SPP |
|---|---|---|---|
| `admin` (id 1) | 511 | 496 | 419 |
| `guru` (id 2) | 1.208 | 194 | 0 |

Dua akun untuk 20 guru.

Masalah kedua yang lebih halus: `absensi`, `nilai`, dan `spp` disimpan memakai
`ON DUPLICATE KEY UPDATE`, yang **tidak menyentuh `created_by` saat baris diperbarui**.
Jadi yang tercatat selalu pembuat pertama, bukan pengubah terakhir — yang Anda minta memang
belum ada kolomnya.

---

## 3. Keadaan sekarang (terverifikasi di produksi)

| Hal | Kondisi |
|---|---|
| Akun aktif | 2 — `admin/admin`, `guru/guru` |
| Master guru | 20 baris |
| Peran tersedia | `admin`, `guru`, `kepala` — belum ada `superadmin` |
| Penautan guru ↔ akun | tidak ada |
| Kolom `updated_by` | **tidak ada di tabel mana pun** |
| `kelas.wali_id` | kolomnya ada, **kosong untuk seluruh 13 kelas**, dan hanya menampung **satu** wali |
| Riwayat perubahan | hanya khusus SPP (`spp_riwayat` + Undo) |
| Sesi login | tidak dilacak — JWT stateless, kedaluwarsa 120 menit |
| FK `created_by` → `users` | **`ON DELETE SET NULL`** di 10 tabel (lihat 4.4) |

---

## 4. Rancangan

### 4.1 Peran

Peran `superadmin` ditambahkan ke enum `users.role`. Peran `kepala` dibiarkan ada di enum
(tidak dipakai, tapi menghapusnya tidak ada untungnya).

| Kemampuan | guru | admin | superadmin |
|---|:--:|:--:|:--:|
| Absensi, nilai | ✓ | ✓ | ✓ |
| SPP | — | ✓ | ✓ |
| Master data (santri, kelas, mapel) | — | ✓ | ✓ |
| Ganti password **sendiri** | ✓ | ✓ | ✓ |
| Buat/ubah/nonaktifkan akun **guru** | — | ✓ | ✓ |
| Buat/ubah/hapus akun **admin & superadmin** | — | — | ✓ |
| Reset password & lihat status password orang lain | — | — | ✓ |
| Tetapkan wali kelas | — | ✓ | ✓ |
| Putuskan sesi orang lain | — | — | ✓ |
| Halaman Aktivitas | — | lihat saja | penuh |

### 4.2 Daftar akun

Gelar `U ` dan kata sambung `BIN` dibuang, format `nama.akhir`, huruf kecil.
Sudah dicek: **tidak ada username yang bentrok**.

| # | Nama di master | Username | Peran | Wali kelas |
|---|---|---|---|---|
| 1 | ACHMAD BIN BIN ALI ASSEGAF | `achmad.assegaf` | guru | **Kelas 1** |
| 2 | SANI BIN AGIL — *"U. Idrus Tsani bin Agil"* | `sani.agil` | guru | **Kelas 2** |
| 3 | U IDRUS BIN AGIL | `idrus.agil` | guru | — |
| 4 | U SHOLEH ASSEGAF | `sholeh.assegaf` | **superadmin** | — |
| 5 | U HUSIN BAAGIL | `husin.baagil` | guru | — |
| 6 | U ALWI ASSEGAF | `alwi.assegaf` | guru | — |
| 7 | U ZAMRONI | `zamroni` | guru | — |
| 8 | U ARIF RAHMAN | `arif.rahman` | guru | — |
| 9 | U ALI AL KAFF | `ali.alkaff` | guru | — |
| 10 | U ALWI AL KAFF | `alwi.alkaff` | guru | — |
| 11 | U IRHAS SOLTHONI | `irhas.solthoni` | guru | — |
| 12 | U ISMAIL AL HASNI | `ismail.alhasni` | guru | — |
| 13 | U SALIM BSA | `salim.bsa` | guru | — |
| 14 | U MUHAMMAD AL HAMID | `muhammad.alhamid` | guru | — |
| 15 | ABUBAKAR MAULADAWILAH | `abubakar.mauladawilah` | guru | **Kelas 6** |
| 16 | NIZAR | `nizar` | guru | **Kelas 2** |
| 17 | ALWY AL IDRUS | `alwy.alidrus` | **superadmin** | **Kelas 5** |
| 18 | MUHAMMAD MASHUR | `muhammad.mashur` | **superadmin** | **Kelas 4** |
| 19 | MUHAMMAD AL HADDAD | `muhammad.alhaddad` | guru | **Kelas 3** |
| 20 | ROMLI | `romli` | guru | **Kelas 1** |

Pembuatannya lewat endpoint admin `POST /users/dari-guru`, bukan migrasi SQL — password harus
di-hash bcrypt dan itu tidak bisa dilakukan di dalam `.sql`. Endpointnya **idempoten**: hanya
membuat akun untuk guru yang belum punya, jadi aman dipanggil ulang saat ada guru baru.

**Akun lama** `admin` (id 1) dan `guru` (id 2) dinonaktifkan setelah akun baru dibagikan.

### 4.3 Wali kelas — butuh tabel baru

`kelas.wali_id` hanya menampung **satu** wali, sedangkan Kelas 1 dan Kelas 2 punya **dua**.
Jadi kolom itu tidak cukup.

Tabel baru **`kelas_wali`** (banyak-ke-banyak):

```sql
CREATE TABLE kelas_wali (
  kelas_id BIGINT  NOT NULL,
  user_id  BIGINT  NOT NULL,
  urutan   TINYINT NOT NULL DEFAULT 1,   -- 1 = wali utama
  PRIMARY KEY (kelas_id, user_id),
  FOREIGN KEY (kelas_id) REFERENCES kelas(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id)  REFERENCES users(id) ON DELETE CASCADE
);
```

`kelas.wali_id` **dihapus** supaya tidak ada dua sumber kebenaran. Aman: kolom itu `NULL`
untuk seluruh 13 kelas, jadi tidak ada data yang hilang.

Penetapan awal:

| Kelas | Santri aktif | Wali |
|---|---:|---|
| Kelas 1 | 17 | `romli`, `achmad.assegaf` |
| Kelas 2 | 25 | `sani.agil`, `nizar` |
| Kelas 3 | 10 | `muhammad.alhaddad` |
| Kelas 4 | 11 | `muhammad.mashur` |
| Kelas 5 | 17 | `alwy.alidrus` |
| Kelas 6 | 6 | `abubakar.mauladawilah` |
| Kelas 2 Pagi | 4 | *sengaja dikosongkan dulu* |
| Kelas 4 Pagi | 3 | *sengaja dikosongkan dulu* |
| Kelas 1 Pagi | 0 | — |
| Sifr A, Sifr B | 0 | — |

**Kelas tanpa wali tetap harus diingatkan.** Kelas 2 Pagi dan Kelas 4 Pagi punya santri aktif
dan ikut masuk pengingat absensi. Supaya tidak hilang diam-diam saat pengingat mulai tertuju
per orang (tahap 4), aturannya:

> Kelas yang **punya** wali → pengingat dikirim ke wali-walinya.
> Kelas yang **belum punya** wali → pengingat tetap dikirim ke **grup**, seperti sekarang.

Dengan begitu tidak ada kelas yang luput hanya karena walinya belum ditetapkan, dan begitu
walinya diisi nanti, pengingatnya otomatis berpindah jadi tertuju — tanpa ubah kode.

### 4.4 Password

Password awal `guru123`, tanpa kewajiban ganti. Guru bisa menggantinya sendiri dari halaman
user.

**Tentang "superadmin bisa melihat password":** password tidak disimpan di database — yang
disimpan **hash bcrypt**, hasil perhitungan satu arah yang tidak bisa dikembalikan. Itu bukan
keterbatasan kode kita; justru itu gunanya, supaya database atau berkas backup yang bocor
tidak membocorkan password.

Usulan yang menutup kebutuhannya tanpa menyimpan password terbaca:

- **Kolom status password.** bcrypt tidak bisa membalik hash, tapi **bisa memeriksa tebakan**.
  Jadi daftar user bisa menampilkan `guru123 (default)` atau `Sudah diganti sendiri`. Karena
  tidak ada kewajiban ganti, hampir semua akan tampil `guru123` — praktisnya Anda melihat
  password mereka.
- **Reset password.** Superadmin mengatur ulang password siapa pun, nilainya ditampilkan
  sekali untuk diberitahukan ke yang bersangkutan.

**Disetujui** — password tidak disimpan terbaca; dipakai status + reset.

**Tombol lihat password di halaman login.** Password awal dibagikan lisan, dan di HP satu huruf
salah ketik tidak kelihatan sama sekali. Ikonnya mata di ujung kanan kolom password; bawaannya
tetap tersembunyi.

### 4.4b Penguncian setelah salah password berkali-kali

Yang dipakai sampai Tahap 4 adalah pembatas laju per **alamat IP**: 5 percobaan / 15 menit,
menghitung **semua** percobaan termasuk yang berhasil. Dua akibatnya baru terlihat setelah
seluruh guru punya akun sendiri:

1. Satu WiFi madrasah = satu alamat IP di mata server. Guru keenam yang login pagi itu ditolak
   walaupun passwordnya benar.
2. Login yang benar pun menghabiskan jatah, jadi yang terkunci justru pengguna yang sah.

Sekarang (`backend/internal/gembok`):

| | Batas | Alasan |
|---|---|---|
| Per nama akun | 5 gagal / 15 menit | menghalangi penebakan password satu orang |
| Per alamat IP | 20 gagal / 15 menit | jaring pengaman untuk pemindai yang mencoba banyak akun |

- Yang dihitung **hanya percobaan gagal**; login yang berhasil menghapus kedua hitungan.
- Pesan salah password menyebutkan sisa kesempatan mulai dari dua terakhir, supaya orang berhenti
  menebak sebelum terkunci.
- Terkunci tercatat di Aktivitas sebagai `login_terkunci`, dibuka admin sebagai `buka_blokir_login`.
- **Admin membuka kunci** dari halaman User (tombol *Buka kunci*, juga terpasang otomatis pada
  *Reset pw*). Membuka sebuah akun sekaligus membuka alamat yang ikut terkunci karenanya —
  kalau tidak, orangnya tetap tertolak dan mengira tombolnya tidak berfungsi.
- Hitungannya **di memori**, bukan database: restart layanan = semua kunci terbuka. Untuk satu
  server madrasah itu justru menyederhanakan — tidak ada yang perlu dibersihkan.

Konsekuensi yang disengaja: orang lain bisa mengunci sebuah akun dengan sengaja salah password
5 kali. Itu melekat pada semua penguncian berbasis akun; ruginya dibatasi karena kuncinya lepas
sendiri dalam 15 menit dan admin bisa membukanya seketika.

### 4.5 Menghapus akun — ada jebakan

Seluruh 10 foreign key `created_by` → `users` memakai **`ON DELETE SET NULL`**. Menghapus akun
secara permanen akan **mengosongkan jejak pembuatnya** tanpa peringatan.

- **Nonaktifkan (bawaan).** `is_active = 0`. Tidak bisa login, riwayat utuh.
- **Hapus permanen.** Hanya untuk akun yang **belum pernah membuat data**. Kalau sudah punya
  jejak, sistem menolak dan menyebutkan berapa baris yang akan terpengaruh.

**Pengaman wajib:** tidak boleh menonaktifkan/menghapus **superadmin terakhir**, dan akun
tidak boleh menghapus dirinya sendiri. Tanpa ini, satu salah klik bisa mengunci semua orang
di luar sistem secara permanen.

### 4.6 Pencatatan perubahan — dua lapis

**Lapis 1 — kolom `updated_by`** (*"siapa terakhir mengubah baris ini?"*), ditambahkan ke:
`absensi, nilai, spp, santri, kelas, mata_pelajaran, periode, users, guru, kelas_mapel, hari_libur`.
Diisi di setiap `UPDATE` **dan** di `ON DUPLICATE KEY UPDATE` — inilah perbaikan atas masalah
di bagian 2. Ditampilkan di layar terkait: *"terakhir diubah U Alwi Assegaf, 18 Agt 16:40"*.

**Lapis 2 — tabel `log_aktivitas`** (*"apa saja yang berubah di sistem?"*), append-only:
pelaku (id + username + nama disalin), `aksi`, `entitas`, `entitas_id`, `ringkasan` siap baca,
`rincian` JSON opsional, `ip`, `created_at`.

**Dicatat per aksi, bukan per baris.** Menyimpan absensi satu kelas menyentuh 10 baris tapi
menghasilkan **satu** entri. Kalau per baris, lognya ~57 entri/hari hanya dari absensi dan
tidak akan ada yang membacanya. Dengan cara ini sekitar 15–30 entri/hari.

`spp_riwayat` **tetap dipertahankan** (fitur Undo bergantung padanya); `log_aktivitas` hanya
menambah satu baris ringkasan per batch. Retensi 2 tahun.

### 4.7 Sesi & riwayat login

- **`sesi_login`** — `user_id`, `token_id`, `ip`, `user_agent`, `dibuat_at`, `terakhir_aktif`,
  `logout_at`. Middleware memeriksa sesi masih hidup di tiap permintaan (satu pembacaan
  ber-indeks, murah untuk 20 pengguna) sehingga **logout benar-benar mencabut token** dan
  superadmin bisa memutus sesi orang lain. `terakhir_aktif` diperbarui maksimal 1×/menit.
- **`log_login`** — percobaan berhasil **dan gagal**. Yang gagal sama pentingnya: sudah ada
  pembatas 5 percobaan/15 menit, dan lonjakan kegagalan adalah tanda pertama ada yang menebak
  password — risiko nyata selama password masih `guru123`.

### 4.8 Halaman admin baru: **Aktivitas**

1. **Sesi aktif** — nama, peran, sejak, terakhir aktif, IP, perangkat, tombol *Putuskan sesi*.
2. **Riwayat login** — berhasil/gagal, waktu, IP.
3. **Log aktivitas** — lini masa, disaring per pengguna / tanggal / jenis aksi.

---

## 5. Alert lewat notifikasi Android

**Telegram yang sekarang sudah berupa notifikasi Android.** Pesan bot masuk sebagai notifikasi
di HP begitu Telegram terpasang dan gurunya ada di grup — sudah berjalan hari ini.

**Peningkatannya: notifikasi per orang.** Kolom `users.telegram_user_id` sudah tersedia sejak
migrasi 007 dan belum pernah dipakai. Setelah akun guru dan `kelas_wali` terisi, pengingatnya
jadi tertuju:

> *"Ustadz Romli, Kelas 1 belum diabsen hari ini."* — dikirim ke Romli **dan** Achmad Assegaf
> saja, bukan ke 20 orang di grup.

Guru cukup chat pribadi ke bot sekali agar `telegram_user_id`-nya terekam.

**Alternatif: PWA + Web Push.** Situs dipasang ke layar utama Android seperti aplikasi dan
mengirim notifikasi sendiri — tanpa Play Store, tanpa biaya. Butuh `manifest.json`, service
worker, kunci VAPID, dan tabel langganan push. Kelebihannya notifikasi terikat akun dan saat
diketuk langsung membuka halaman absensi kelasnya.

Satu hal yang perlu diketahui sejak awal: di HP Xiaomi, Oppo, Vivo, Realme dengan penghemat
baterai agresif, proses latar browser sering dimatikan sistem sehingga **notifikasi PWA bisa
terlambat atau tidak muncul**. Telegram, sebagai aplikasi biasa, jauh lebih andal di HP
seperti itu — dan merek-merek tersebut sangat umum dipakai. Untuk iPhone, PWA push baru
bekerja di iOS 16.4+ dan hanya bila situsnya sudah ditambahkan ke layar utama.

**Aplikasi Android asli** tidak sebanding: akun Play Store berbayar, alur rilis, pemeliharaan
versi — untuk manfaat yang hampir sama.

**Saran: Telegram per orang dulu, PWA menyusul bila perlu.** Keduanya bisa berdampingan.

---

## 6. Perubahan yang diperlukan

### Database

**`020_akun_guru.sql`**
```sql
ALTER TABLE users
  MODIFY COLUMN role ENUM('superadmin','admin','guru','kepala') NOT NULL DEFAULT 'guru',
  ADD COLUMN guru_id BIGINT NULL UNIQUE AFTER role,
  ADD COLUMN terakhir_login DATETIME NULL,
  ADD CONSTRAINT fk_users_guru FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE SET NULL;
```

**`021_wali_kelas.sql`** — tabel `kelas_wali`, lalu `ALTER TABLE kelas DROP COLUMN wali_id`.

**`022_audit.sql`** — tabel `log_aktivitas`, `sesi_login`, `log_login`, plus kolom
`updated_by` di 11 tabel pada 4.6.

### Backend

| Berkas | Perubahan |
|---|---|
| `handlers/users.go` | `POST /users/dari-guru`, reset password, status password, ganti password sendiri, pengaman hapus (4.5) |
| `handlers/auth.go` | catat login berhasil/gagal, buat & hapus sesi |
| `middleware/auth.go` | validasi sesi + perbarui `terakhir_aktif` |
| `middleware/` | penjaga peran `superadmin` |
| `handlers/master_crud.go` | CRUD wali kelas (`kelas_wali`) |
| **`internal/audit`** (paket baru) | `audit.Catat(...)` agar format ringkasan seragam |
| `handlers/absensi.go`, `nilai.go`, `spp.go`, `master_crud.go` | isi `updated_by` + `audit.Catat` |
| `handlers/aktivitas.go` (baru) | endpoint tiga tab halaman Aktivitas |
| `notifworker/pengingat.go` | pengingat absensi menyebut nama wali kelas |

### Frontend

- Halaman **Aktivitas** (tiga tab).
- **Master → User**: tombol "Buatkan akun dari master guru", kolom status password, tombol
  Reset & Ganti password, pengaman hapus.
- **Master → Kelas**: pemilihan wali kelas (boleh lebih dari satu).
- Menu **Ganti password saya** untuk semua peran.
- "Terakhir diubah oleh …" di halaman absensi, nilai, dan SPP.

---

## 7. Risiko

| Risiko | Penanganan |
|---|---|
| Guru bingung dengan login baru | akun lama dinonaktifkan **setelah** akun baru dibagikan; sediakan daftar cetak username |
| `guru123` sama untuk semua, tidak wajib diganti | pilihan Anda; halaman Aktivitas diberi keterangan bahwa identitas belum terjamin selama password masih default |
| Hapus akun menghancurkan jejak | `ON DELETE SET NULL` — hapus permanen dibatasi (4.5) |
| Terkunci di luar sistem | superadmin terakhir tidak bisa dinonaktifkan/dihapus |
| Log membengkak | dicatat per aksi, dipangkas setelah 2 tahun |

Data lama **tidak diubah**. Tidak ada upaya menebak siapa sebenarnya pengisi 1.489 absensi
terdahulu — menebak justru akan mengotori audit.

---

## 8. Tahapan pengerjaan

| Tahap | Isi | Hasil |
|---|---|---|
| **1** | peran superadmin, 20 akun, wali kelas, ganti/reset password, status password, pengaman hapus | tiap orang punya identitas sendiri |
| **2** | `updated_by` + `log_aktivitas` | perubahan mulai terekam |
| **3** | sesi login + `log_login` + halaman Aktivitas | pemantauan lengkap |
| **4** | Telegram per orang (`telegram_user_id` + `kelas_wali`) | alert Android tertuju per wali kelas |

---

## 9. Sisa pertanyaan

**Sudah dijawab:** gelar `U ` = Ustadz (dibuang) · peran `kepala` tidak dipakai · Kelas 2 Pagi
& Kelas 4 Pagi dikosongkan dulu · wali Kelas 2 = #2 `SANI BIN AGIL` · password memakai
status + reset.

**Tersisa satu, dengan asumsi bawaan yang sudah dipakai di rancangan ini:**

> **Akun superadmin terpisah** untuk Muhammad Al Masyhur, Alwy Alaydrus, dan Sholeh Assegaf —
> bukan satu akun `admin` bersama. Tanpa ini wali Kelas 4 dan Kelas 5 tidak bisa dibedakan
> (lihat 1.1). Bilang saja kalau Anda tetap memilih satu akun bersama.

### Catatan ejaan nama

Ejaan di master berbeda dengan yang Anda tulis: `SANI BIN AGIL` vs "Idrus Tsani bin Agil",
`ALWY AL IDRUS` vs "Alwy Alaydrus", `MUHAMMAD MASHUR` vs "Muhammad Al Masyhur",
`ABUBAKAR MAULADAWILAH` vs "Abubakar Mauladdawilah", dan `ACHMAD BIN BIN ALI ASSEGAF` yang
"BIN"-nya dobel.

Bawaannya **master dibiarkan apa adanya** dan username mengikuti ejaan sekarang. Konsekuensi
yang perlu diketahui: wali Kelas 2 akan menerima username **`sani.agil`**, padahal beliau
dikenal sebagai "Ustadz Idrus Tsani bin Agil" — bisa membingungkan saat kredensialnya
dibagikan.

Kalau ejaan #2 dirapikan jadi "IDRUS TSANI BIN AGIL", usernamenya akan **bentrok** dengan #3
(`U IDRUS BIN AGIL`) yang sama-sama menghasilkan `idrus.agil`. Penyelesaiannya memakai nama
tengah: #2 → `idrus.tsani`, #3 → `idrus.agil`. Kalau mau begitu, tinggal bilang.
