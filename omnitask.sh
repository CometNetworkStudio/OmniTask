#!/usr/bin/env bash
# OmniTask 引导式工具箱（Linux/macOS）
# 用法：./omnitask.sh  然后按提示输入选项
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD="$ROOT/build"
WEB="$ROOT/backend/internal/server/webdist"

build_frontend() {
  local base="${VITE_BASE:-/}"
  echo "==> 构建前端 (base=$base)"
  ( cd "$ROOT/frontend" && VITE_BASE="$base" npm run build )
  mkdir -p "$WEB"
  find "$WEB" -mindepth 1 ! -name '.gitkeep' -delete
  cp -r "$ROOT/frontend/dist/." "$WEB/"
}

# build_go <server|agent|migrate> <linux|windows|both>
build_go() {
  local cmd="$1" target="$2"
  local name="$cmd"
  local out
  case "$cmd" in
    server) out="omnitask-server" ;;
    agent) out="omnitask-agent" ;;
    migrate) out="omnitask-migrate" ;;
    *) echo "未知命令: $cmd"; return 1 ;;
  esac
  mkdir -p "$BUILD"
  ( cd "$ROOT/backend"
    case "$target" in
      linux)
        echo "==> 编译 $out (linux/amd64)"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$BUILD/${out}-linux-amd64" "./cmd/${cmd}" ;;
      windows)
        echo "==> 编译 $out (windows/amd64)"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o "$BUILD/${out}-windows-amd64.exe" "./cmd/${cmd}" ;;
      both)
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$BUILD/${out}-linux-amd64" "./cmd/${cmd}"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o "$BUILD/${out}-windows-amd64.exe" "./cmd/${cmd}"
        echo "==> 编译 $out (linux+windows/amd64) 完成" ;;
      *) echo "未知目标: $target"; return 1 ;;
    esac
  )
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
  build_go migrate "$target"
  echo "构建产物目录：$BUILD"
  ls -1 "$BUILD"
}

menu() {
  echo
  echo "================ OmniTask 工具箱 ================"
  echo " 1) 部署（构建前端+后端+Host，迁移，重启）[Linux]"
  echo " 2) 构建后端 server（选择目标平台）"
  echo " 3) 构建脚本 Host agent（选择目标平台）"
  echo " 4) 构建迁移工具 migrate（选择目标平台）"
  echo " 5) 构建前端（dist，可内嵌进后端）"
  echo " 6) 构建全部（前端+server+agent+migrate，选择目标平台）"
  echo " 0) 退出"
  echo "================================================="
}

while true; do
  menu
  read -r -p "请输入选项数字: " choice
  case "${choice:-}" in
    1) "$ROOT/deploy.sh" ;;
    2) build_go server "$(ask_target)" ;;
    3) build_go agent "$(ask_target)" ;;
    4) build_go migrate "$(ask_target)" ;;
    5) build_frontend ;;
    6) build_all "$(ask_target)" ;;
    0) echo "再见"; exit 0 ;;
    *) echo "无效选项，请重新输入。" ;;
  esac
done
