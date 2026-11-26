# Full 模式 Docker Compose

一条命令启动：MySQL、Redis、server、Host（agent）、nginx。

## 启动

```bash
# 可选：复制端口/密码/镜像源配置
cp deploy/docker-compose/.env.example deploy/docker-compose/.env

docker compose --env-file deploy/docker-compose/.env \
  -f deploy/docker-compose/docker-compose.full.yml up -d --build
```

不需要自定义时：

```bash
docker compose -f deploy/docker-compose/docker-compose.full.yml up -d --build
```

默认已使用 `registry.npmmirror.com` 与 `goproxy.cn`。海外环境可覆盖：

```bash
NPM_REGISTRY=https://registry.npmjs.org \
GOPROXY=https://proxy.golang.org,direct \
docker compose -f deploy/docker-compose/docker-compose.full.yml up -d --build
```

默认只监听回环地址：`http://127.0.0.1:4096/`。生产环境由宿主机 nginx 反代并终结 HTTPS；本地需要从其他机器直连时，把 `.env` 里的 `OMNITASK_BIND` 改为 `0.0.0.0`，并自行限制防火墙。

首个注册用户自动成为管理员。正式部署建议关闭该行为，使用 `OMNITASK_ADMIN_USER` / `OMNITASK_ADMIN_PASSWORD` 预创建管理员。之后在「脚本」页安装 cidaren / welearn 等脚本。

## 端口

- 宿主机回环：`127.0.0.1:4096`（`OMNITASK_BIND` / `OMNITASK_PORT` 可改）
- 容器内：server 8080/9000，agent 50051，MySQL 3306，Redis 6379
- MySQL/Redis 默认不暴露到宿主机

## 配置与数据

- 配置模板：`config.full.yml`、`config.agent.yml`
- 生产环境建议复制 `config.full.yml` 为 `config.prod.yml`，填好数据库密码、主密钥、邀请码后，在 `.env` 设置 `OMNITASK_CONFIG_TEMPLATE=./config.prod.yml`；该文件已 gitignore
- 首次启动复制到 Docker 卷 `omnitask-server-config` / `omnitask-agent-config`，之后只修改卷内文件
- 脚本、日志、任务工作区和跨进程限速库在共享卷 `omnitask-data` 的 `/data/scripts`、`/data/logs`、`/data/workspaces`、`/data/rate`
- 修改配置（先把卷内配置复制出来编辑，再写回）：
  ```bash
  docker compose -f deploy/docker-compose/docker-compose.full.yml \
    cp omnitask-server:/config/config.yaml /tmp/omnitask-config.yaml
  # 编辑 /tmp/omnitask-config.yaml
  docker compose -f deploy/docker-compose/docker-compose.full.yml \
    cp /tmp/omnitask-config.yaml omnitask-server:/config/config.yaml
  docker compose -f deploy/docker-compose/docker-compose.full.yml restart omnitask-server
  ```
- 重置配置卷（会丢失主密钥，已加密凭据将无法解密，慎用）：
  ```bash
  docker compose -f deploy/docker-compose/docker-compose.full.yml down
  docker volume rm omnitask_omnitask-server-config omnitask_omnitask-agent-config
  ```

## 更新

```bash
git pull
docker compose -f deploy/docker-compose/docker-compose.full.yml up -d --build
```

## 注意

- 如果本机已经用 `backend/deploy/docker-compose` 起过 `omnitask-db` / `omnitask-redis`，先停掉，避免容器名与端口冲突
- 生产环境请修改 MySQL 密码，并同步修改 `config.full.yml` 里的 `data.db.user.dsn`
- 当前 compose 是单机多进程；Host 与 server 共享 `/data/scripts` 卷
- 账号级代理在「凭据」页配置；脚本通过 `OMNITASK_PROXY` 自动读取，SDK 的 `new_session()` 已内置
- 脚本依赖在 server 容器内安装；国内可把 `agent.pip_index_url` 设为 `https://pypi.tuna.tsinghua.edu.cn/simple`
- 任务工作区默认保留失败任务，长期运行可用 `agent.keep_failed_workspaces: false` 关闭
- 如需多机 Host，需要额外的节点注册、脚本分发与账号亲和调度，见 `docs/context.md`
- 需要 HTTPS 时，把证书挂载到 nginx 并修改 `nginx.conf`
