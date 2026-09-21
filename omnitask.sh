#!/usr/bin/env bash
# OmniTask 引导式工具箱（Linux/macOS）
# 用法：
#   ./omnitask.sh                     # 进入数字菜单
#   ./omnitask.sh deploy              # 非交互：部署
#   ./omnitask.sh server|agent [linux|windows|both]
#   ./omnitask.sh frontend [base]     # base 默认 /
#   ./omnitask.sh all [linux|windows|both]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD="$ROOT/build"
WEB="$ROOT/backend/internal/server/webdist"
RUN="${OMNITASK_RUN:-/tmp/opencode}"
PID_BACKEND="$RUN/omnitask-backend.pid"
PID_AGENT="$RUN/omnitask-agent.pid"
GOCMD=(go build -trimpath -ldflags "-s -w")   # -s -w 去掉符号表/调试信息，减小体积
mkdir -p "$BUILD" "$RUN"

# build_frontend [base]
build_frontend() {
  local base="${1:-${VITE_BASE:-/}}"
  echo "==> 构建前端 (base=$base)"
  ( cd "$ROOT/frontend" && VITE_BASE="$base" npm run build )
  mkdir -p "$WEB"
  find "$WEB" -mindepth 1 ! -name '.gitkeep' -delete
  cp -r "$ROOT/frontend/dist/." "$WEB/"
}

# build_go <server|agent> <linux|windows|both>
build_go() {
  local cmd="$1" target="$2" out
  case "$cmd" in
    server) out="omnitask-server" ;;
    agent) out="omnitask-agent" ;;
    *) echo "未知命令: $cmd"; return 1 ;;
  esac
  ( cd "$ROOT/backend"
    case "$target" in
      linux)
        echo "==> 编译 $out (linux/amd64)"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "${GOCMD[@]}" -o "$BUILD/${out}-linux-amd64" "./cmd/${cmd}" ;;
      windows)
        echo "==> 编译 $out (windows/amd64)"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "${GOCMD[@]}" -o "$BUILD/${out}-windows-amd64.exe" "./cmd/${cmd}" ;;
      both)
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "${GOCMD[@]}" -o "$BUILD/${out}-linux-amd64" "./cmd/${cmd}"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "${GOCMD[@]}" -o "$BUILD/${out}-windows-amd64.exe" "./cmd/${cmd}"
        echo "==> 编译 $out (linux+windows/amd64) 完成" ;;
      *) echo "未知目标: $target"; return 1 ;;
    esac
  )
}

# 当前系统原生编译到指定路径
build_native() {
  ( cd "$ROOT/backend" && "${GOCMD[@]}" -o "$2" "./cmd/$1" )
}

ask_target() {
  {
    echo "选择目标平台："
    echo "  1) linux-amd64"
    echo "  2) windows-amd64"
    echo "  3) 两者都编译"
  } >&2
  read -r -p "输入 1/2/3 [默认 1]: " t
  case "${t:-1}" in
    2) echo windows ;;
    3) echo both ;;
    *) echo linux ;;
  esac
}

build_all() {
  local target="$1"
  build_frontend
  build_go server "$target"
  build_go agent "$target"
  echo "构建产物目录：$BUILD"
  ls -1 "$BUILD"
}

# 自检：后端 build/vet/test + 前端构建 + SDK 测试
check() {
  echo "==> 后端 build/vet/test"
  ( cd "$ROOT/backend" && go build ./... && go vet ./... && go test ./... )
  echo "==> 前端 build"
  ( cd "$ROOT/frontend" && npm run build )
  echo "==> SDK 测试"
  ( cd "$ROOT/sdk/python" && ./.venv/bin/python -m unittest discover -s tests -t . )
  echo "==> 全部检查通过"
}

stop_by_pidfile() {
  local f="$1"
  if [ -f "$f" ]; then
    kill "$(cat "$f")" 2>/dev/null || true
    sleep 1
    rm -f "$f"
  fi
}

# start_bg <pidfile> <logfile> <cmd...>：在 backend/ 目录下后台启动（相对路径如日志/脚本目录以 backend 为基准）
start_bg() {
  local pidf="$1" logf="$2"
  shift 2
  ( cd "$ROOT/backend" && exec setsid "$@" < /dev/null > "$logf" 2>&1 ) &
  echo $! > "$pidf"
}

# 部署：构建前端(base=/omnitask/) + 后端/Host(原生) + 迁移 + 重启
deploy() {
  echo "==> 部署"
  build_frontend /omnitask/
  echo "==> 编译 backend / agent（原生）"
  build_native server "$RUN/omnitask-server"
  build_native agent "$RUN/omnitask-agent"

  echo "==> 迁移"
  set -a
  # shellcheck disable=SC1091
  . "$ROOT/.env"
  set +a
  CONF="$ROOT/backend/config/deploy.yml"
  ( cd "$ROOT/backend" && "$RUN/omnitask-server" -migrate -conf "$CONF" )

  echo "==> 重启 backend"
  stop_by_pidfile "$PID_BACKEND"
  start_bg "$PID_BACKEND" "$RUN/backend.log" "$RUN/omnitask-server" -conf "$CONF"

  echo "==> 重启 agent"
  stop_by_pidfile "$PID_AGENT"
  start_bg "$PID_AGENT" "$RUN/agent.log" "$RUN/omnitask-agent" -conf "$CONF"

  sleep 2
  echo "==> 完成"
}

menu() {
  echo
  echo "================ OmniTask 工具箱 ================"
  echo " 1) 部署（构建前端+后端+Host，迁移，重启）"
  echo " 2) 构建后端 server（选择目标平台）"
  echo " 3) 构建脚本 Host agent（选择目标平台）"
  echo " 4) 构建前端（dist，可内嵌进后端）"
  echo " 5) 构建全部（前端+server+agent，选择目标平台）"
  echo " 6) 自检（后端 build/vet/test + 前端构建 + SDK 测试）"
  echo " 0) 退出"
  echo "================================================="
}

# 非交互模式
if [ -n "${1:-}" ]; then
  case "$1" in
    deploy) deploy ;;
    server) build_go server "${2:-linux}" ;;
    agent) build_go agent "${2:-linux}" ;;
    frontend) build_frontend "${2:-}" ;;
    all) build_all "${2:-both}" ;;
    check) check ;;
    *) echo "未知命令: $1"; exit 1 ;;
  esac
  exit 0
fi

# 交互菜单
while true; do
  menu
  read -r -p "请输入选项数字: " choice
  case "${choice:-}" in
    1) deploy ;;
    2) build_go server "$(ask_target)" ;;
    3) build_go agent "$(ask_target)" ;;
    4) build_frontend ;;
    5) build_all "$(ask_target)" ;;
    6) check ;;
    0) echo "再见"; exit 0 ;;
    *) echo "无效选项，请重新输入。" ;;
  esac
done
