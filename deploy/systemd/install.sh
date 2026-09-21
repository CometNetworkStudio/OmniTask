#!/usr/bin/env bash
# 安装 OmniTask 的 systemd 服务（服务器重启自动拉起）。
# 用法：sudo ./deploy/systemd/install.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
UNIT_DIR=/etc/systemd/system

for name in omnitask-server omnitask-agent; do
  sed "s#/home/OmniTask#${ROOT}#g" "$ROOT/deploy/systemd/${name}.service" > "${UNIT_DIR}/${name}.service"
  echo "installed ${UNIT_DIR}/${name}.service"
done

systemctl daemon-reload
systemctl enable omnitask-server omnitask-agent
echo "已安装并设为开机自启。"
echo "启动/重启：sudo systemctl restart omnitask-server omnitask-agent"
echo "查看日志：journalctl -u omnitask-server -f"
