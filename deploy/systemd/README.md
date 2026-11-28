# systemd 服务

`omnitask-server.service` / `omnitask-host.service` 让后端与脚本 Host 作为系统服务运行，
**服务器重启后自动拉起**（此前由 `omnitask.sh deploy` 以 `setsid` 启动，重启不会恢复）。

## 安装

```bash
./omnitask.sh deploy      # 先构建产物到 bin/ 并跑一次（生成 config）
sudo ./deploy/systemd/install.sh
sudo systemctl start omnitask-server omnitask-host
```

安装脚本会把模板里的 `/home/OmniTask` 替换为当前仓库路径。

## 常用命令

```bash
sudo systemctl restart omnitask-server omnitask-host
systemctl status omnitask-server
journalctl -u omnitask-server -f
```

> 安装系统服务后，`omnitask.sh deploy` 会自动改用 `systemctl restart`（检测到 unit 文件时）。
