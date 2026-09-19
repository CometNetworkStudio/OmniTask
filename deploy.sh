#!/usr/bin/env bash
# OmniTask 一键部署脚本：构建前端/后端 -> 迁移 -> 重启 backend/agent
# 目标地址：https://<your-host>:4096/omnitask/
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN="/tmp/opencode"
WEB="/var/www/omnitask"
PID_BACKEND="$RUN/omnitask-backend.pid"
PID_AGENT="$RUN/omnitask-agent.pid"
mkdir -p "$RUN"

# 清理可能残留的进程（无 pidfile 时也能幂等）
pkill -x omnitask-server 2>/dev/null || true
pkill -f 'python -u grpc_server.py' 2>/dev/null || true

stop_by_pidfile() {
  local f="$1"
  if [ -f "$f" ]; then
    kill "$(cat "$f")" 2>/dev/null || true
    sleep 1
    rm -f "$f"
  fi
}

echo "==> build frontend (base=/omnitask/)"
cd "$ROOT/frontend"
VITE_BASE=/omnitask/ npm run build
rm -rf "$WEB"
mkdir -p "$WEB"
cp -r dist/* "$WEB/"

echo "==> build backend"
cd "$ROOT/backend"
go build -o "$RUN/omnitask-server" ./cmd/server

echo "==> migrate"
set -a
# shellcheck disable=SC1091
. "$ROOT/.env"
set +a
go run ./cmd/migration -conf config/deploy.yml > /dev/null 2>&1 || {
  echo "migration failed"; exit 1;
}

echo "==> restart backend"
stop_by_pidfile "$PID_BACKEND"
setsid "$RUN/omnitask-server" -conf config/deploy.yml < /dev/null > "$RUN/backend.log" 2>&1 &
echo $! > "$PID_BACKEND"

echo "==> restart agent"
stop_by_pidfile "$PID_AGENT"
cd "$ROOT/agent"
OMNITASK_BANK_MODE=grpc setsid ./.venv/bin/python -u grpc_server.py < /dev/null > "$RUN/agent.log" 2>&1 &
echo $! > "$PID_AGENT"

sleep 2
echo "==> done"
echo "    https://<your-host>:4096/omnitask/"
