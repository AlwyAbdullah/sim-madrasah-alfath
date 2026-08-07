#!/usr/bin/env bash
# Backup harian database SIM-Madrasah.
#
# Dijalankan otomatis oleh systemd timer (sim-madrasah-backup.timer) sebagai root,
# sehingga mysqldump bisa memakai autentikasi socket tanpa menyimpan password.
#
# Yang dilakukan:
#   1. dump database  -> .sql.gz
#   2. VERIFIKASI hasil dump (bukan sekadar membuat file): uji gzip + cek isi
#   3. simpan salinan konfigurasi (.env, nginx, unit systemd) utk pemulihan total
#   4. hapus backup lama sesuai aturan retensi
#
# Keluar dengan kode != 0 bila ada yang gagal, supaya systemd menandainya FAILED
# dan terlihat di `systemctl list-timers` / `journalctl -u sim-madrasah-backup`.
set -euo pipefail

DB="${DB_NAME:-sim_madrasah}"
DIR="${BACKUP_DIR:-/home/deploy/backups}"
PEMILIK="${BACKUP_OWNER:-deploy:deploy}"
MIN_TABEL="${MIN_TABEL:-15}"   # dump dianggap mencurigakan bila tabelnya lebih sedikit dari ini

STAMP="$(date +%Y-%m-%d_%H%M%S)"
FILE="$DIR/${DB}-${STAMP}.sql.gz"

mkdir -p "$DIR"
chown "$PEMILIK" "$DIR" 2>/dev/null || true
chmod 750 "$DIR"

echo ">> [$(date '+%F %T')] mulai backup database '$DB'"

# ---- 1. dump ----
# --single-transaction : konsisten tanpa mengunci tabel (aman saat aplikasi jalan)
# --routines/--triggers/--events : ikut sertakan objek selain tabel
mysqldump \
  --single-transaction \
  --routines --triggers --events \
  --default-character-set=utf8mb4 \
  "$DB" | gzip -9 > "$FILE"

# ---- 2. verifikasi ----
if [[ ! -s "$FILE" ]]; then
  echo "!! GAGAL: berkas backup kosong: $FILE" >&2
  rm -f "$FILE"
  exit 1
fi

if ! gzip -t "$FILE" 2>/dev/null; then
  echo "!! GAGAL: berkas gzip rusak: $FILE" >&2
  rm -f "$FILE"
  exit 1
fi

JML_TABEL="$(zcat "$FILE" | grep -c '^CREATE TABLE' || true)"
if (( JML_TABEL < MIN_TABEL )); then
  echo "!! GAGAL: hanya $JML_TABEL tabel di dump (minimal $MIN_TABEL) — dump tidak dipercaya" >&2
  rm -f "$FILE"
  exit 1
fi

if ! zcat "$FILE" | tail -5 | grep -q 'Dump completed'; then
  echo "!! GAGAL: penanda 'Dump completed' tidak ditemukan — dump terpotong" >&2
  rm -f "$FILE"
  exit 1
fi

chown "$PEMILIK" "$FILE" 2>/dev/null || true
chmod 640 "$FILE"
echo "   OK  $(basename "$FILE")  ($(du -h "$FILE" | cut -f1), $JML_TABEL tabel)"

# ---- 3. salinan konfigurasi (berisi rahasia -> izin ketat) ----
CFG="$DIR/config-${STAMP}.tar.gz"
tar czf "$CFG" \
  -C / \
  --ignore-failed-read \
  home/deploy/sim-madrasah/backend/.env \
  home/deploy/sim-madrasah/frontend/.env.production \
  home/deploy/waha/.env \
  home/deploy/waha/docker-compose.yml \
  etc/nginx/sites-available/sim-madrasah \
  etc/systemd/system/sim-madrasah-backend.service \
  etc/systemd/system/sim-madrasah-frontend.service \
  2>/dev/null || true
if [[ -s "$CFG" ]]; then
  chown "$PEMILIK" "$CFG" 2>/dev/null || true
  chmod 600 "$CFG"   # berisi kata sandi & secret — jangan bisa dibaca umum
  echo "   OK  $(basename "$CFG")  ($(du -h "$CFG" | cut -f1))"
fi

# ---- 4. retensi ----
# Harian disimpan 30 hari. Backup tanggal 1 tiap bulan disimpan 1 tahun.
find "$DIR" -maxdepth 1 -type f -name "${DB}-*.sql.gz" ! -name "*-01_*" -mtime +30 -print -delete | sed 's/^/   hapus (harian >30 hari): /' || true
find "$DIR" -maxdepth 1 -type f -name "${DB}-*-01_*.sql.gz" -mtime +365 -print -delete | sed 's/^/   hapus (bulanan >1 tahun): /' || true
find "$DIR" -maxdepth 1 -type f -name "config-*.tar.gz" -mtime +30 -delete || true

JML="$(find "$DIR" -maxdepth 1 -type f -name "${DB}-*.sql.gz" | wc -l)"
TOTAL="$(du -sh "$DIR" | cut -f1)"
echo ">> selesai — $JML berkas backup, total $TOTAL di $DIR"
