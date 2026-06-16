#!/usr/bin/env bash
# Update SIM-Madrasah ke VPS (jalankan dari mesin lokal, di Git Bash).
# Backend: cross-compile -> kirim binary -> restart. Frontend: kirim source -> build -> restart.
# Migrasi DB baru TIDAK otomatis — jalankan manual bila ada file migrations/ baru.
set -e

KEY="$HOME/.ssh/id_ed25519"
HOST="deploy@103.175.219.47"
REMOTE="/home/deploy/sim-madrasah"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo ">> Backend: cross-compile linux/amd64"
cd "$ROOT/backend"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o sim-server ./cmd/server
echo ">> Kirim binary & restart backend"
# kirim ke nama sementara lalu rename (binary lama sedang running, tak bisa ditimpa in-place)
scp -i "$KEY" sim-server "$HOST:$REMOTE/backend/sim-server.new"
rm -f sim-server
ssh -i "$KEY" "$HOST" "mv -f $REMOTE/backend/sim-server.new $REMOTE/backend/sim-server && chmod +x $REMOTE/backend/sim-server && sudo systemctl restart sim-madrasah-backend && sleep 2 && systemctl is-active sim-madrasah-backend"

echo ">> Frontend: kirim source, build, restart"
cd "$ROOT/frontend"
tar czf - --exclude=node_modules --exclude=.next --exclude=.env.local . \
  | ssh -i "$KEY" "$HOST" "tar xzf - -C $REMOTE/frontend"
ssh -i "$KEY" "$HOST" "cd $REMOTE/frontend && npm ci && npm run build && sudo systemctl restart sim-madrasah-frontend && systemctl is-active sim-madrasah-frontend"

echo ">> Selesai. http://103.175.219.47"
