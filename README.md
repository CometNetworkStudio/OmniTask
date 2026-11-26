![OmniTask 万务通](docs/assets/banner-light.png#gh-light-mode-only)
![OmniTask 万务通](docs/assets/banner-dark.png#gh-dark-mode-only)

# OmniTask 万务通

> 一处托管，万务自达。面向大学生的校园任务自动化聚合平台。

OmniTask 把学习通、词达人、WeLearn 等平台的重复任务交给可插拔脚本执行。用户一次配置，平台负责凭据、任务编排和实时进度，支持官方实例与自托管。

[门户](https://omnitasks.top/) ｜ [在线使用](https://app.omnitasks.top/) ｜ [部署文档](docs/deployment-profiles.md) ｜ [用户协议](https://app.omnitasks.top/terms) ｜ [QQ 频道](https://pd.qq.com/s/N834h64oek)

[![License](https://img.shields.io/badge/license-MIT-2E7D32.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-42B883?logo=vuedotjs&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)

## 痛点

- 大学生要面对四六级、计算机二级、普通话、学科竞赛，以及学校布置的大量线上任务（学习通时长、词达人、WeLearn 作业和挂时长等），重复且耗时。
- 现有开源脚本零散，要装环境、要开 GUI，跑起来就占用一台设备，普通人用不起来。
- 部分人拿这些开源脚本当“信息差”牟利，不公平，而需求真实存在。

## 功能

| 模块 | 说明 |
|------|------|
| 账号与会话 | 服务端会话 Cookie，可随时吊销；角色 `admin` / `user`，Casbin 鉴权 |
| 凭据管理 | AES-256-GCM 加密入库，任务时解密；主密钥可轮换；支持平台级代理和访问审计 |
| 任务中心 | 按平台和动作创建任务，参数由脚本 manifest 动态生成，支持取消、重试和历史 |
| 实时进度 | WebSocket 事件流，页面实时查看步骤、日志和进度 |
| 题库 | 答题结果沉淀，命中直接作答，减少重复调用 |
| 脚本生态 | GitHub 安装、启停、回滚、调试和卸载；脚本独立发布，可单独 CLI 运行 |

## 架构

```text
浏览器（Vue 3 + Vuetify）
        │ REST + WebSocket
        ▼
Go 后端（Gin + GORM + Casbin）── 编排、存储、鉴权、脚本安装
        │ gRPC AgentService
        ▼
Go Host（cmd/agent）── 扫描脚本、启动子进程、转发事件
        │ stdio JSON
        ▼
平台脚本（任意语言，推荐 Python）
```

前端只负责交互，后端只负责编排，Host 管理脚本生命周期，脚本不直接连接数据库。

## 快速开始

### Docker（推荐）

```bash
cp deploy/docker-compose/.env.example deploy/docker-compose/.env
# 修改 .env 中的 MySQL 密码、管理员密码和端口

docker compose --env-file deploy/docker-compose/.env \
  -f deploy/docker-compose/docker-compose.full.yml up -d --build
```

默认访问 `http://127.0.0.1:4096/`。生产部署、HTTPS、域名和备份见 [docs/deployment-profiles.md](docs/deployment-profiles.md)。

### lite（单机）

```bash
cd frontend && npm install && npm run build && cd ..
cd backend
go build -o omnitask-server ./cmd/server
go build -o omnitask-agent  ./cmd/agent
./omnitask-server -conf config.yaml
```

打开 `http://127.0.0.1:8080`。lite 使用 SQLite，无需 MySQL 和 Redis，首次运行会生成 `config.yaml`。

更多构建参数和两种运行模式见 [docs/build-and-run.md](docs/build-and-run.md)。

## 脚本生态

平台能力全部来自可插拔脚本，已有项目可以选择三种接入方式：

- 包装仓：保留原项目，加一个 `adapter.py` 映射 action。
- 装饰器：`from omnitask_sdk import action, serve`，几行代码接入。
- CLI 桥：已有命令行工具，只写 `script.json`。

```bash
pip install "git+https://github.com/CometNetworkStudio/OmniTask.git#subdirectory=sdk/python"
```

官方脚本：[cidaren](https://github.com/CometNetworkStudio/cidaren) ｜ [welearn](https://github.com/CometNetworkStudio/welearn)

脚本协议、接入方式和开发指南见 [docs/script-protocol.md](docs/script-protocol.md)、[docs/integration-guide.md](docs/integration-guide.md)、[docs/script-development.md](docs/script-development.md)。

## 安全与合规

- 会话保存在服务端，Cookie 为 `HttpOnly` + `SameSite=Lax`，非安全方法校验 CSRF。
- 凭据加密存储，日志脱敏，登录失败锁定，脚本子进程只继承白名单环境变量。
- 部署实例在任务运行时技术上可以解密凭据。请只使用你信任的实例，或自行部署。
- 使用规则、隐私政策、数据保留与删除、滥用投诉和封禁流程见 [用户协议](https://app.omnitasks.top/terms) 和 [隐私政策](https://app.omnitasks.top/privacy)。
- 滥用、隐私和账号申诉请加入 [QQ 频道](https://pd.qq.com/s/N834h64oek)，频道号 `pd84113918`。

本项目仅供个人学习和技术研究使用。请遵守各平台服务条款与学校规定，因使用本项目产生的一切后果由使用者自行承担。

## 文档

| 文档 | 内容 |
|------|------|
| [部署模式](docs/deployment-profiles.md) | 自托管、官方托管、Nginx、HTTPS、备份 |
| [构建与运行](docs/build-and-run.md) | lite / full、构建参数、运行方式 |
| [脚本协议](docs/script-protocol.md) | manifest、stdio JSON、事件和运行时 |
| [接入指南](docs/integration-guide.md) | 三种脚本接入方式 |
| [脚本开发](docs/script-development.md) | 组件、manifest v2、本地调试 |
| [上下文记录](docs/context.md) | 关键决策和版本演进 |

## License

[MIT](LICENSE) © CometNetworkStudio
