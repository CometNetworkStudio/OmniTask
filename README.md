# OmniTask 万务通

> 一处托管，万务自达。面向大学生的校园任务自动化聚合平台

OmniTask 把重复、耗时的线上学习任务交给自动化脚本执行：用户一次配置，平台编排任务、脚本并发执行，前端实时查看进度，无需守着电脑或手机。

> 上下文/决策记录：`docs/context.md` ｜ 开发规划：`docs/development-plan.md` ｜ 脚本协议：`docs/script-protocol.md` ｜ 脚本开发与接入：`docs/integration-guide.md`

---

## 痛点

- 大学生要面对四六级、计算机二级、普通话、学科竞赛，以及学校布置的大量线上任务（学习通时长、词达人、WeLearn 作业/挂时长等），重复且耗时。
- 现有开源脚本零散、要装环境、要开 GUI、跑起来就占用一台设备，普通人用不起来。
- 部分人拿这些开源脚本当"信息差"牟利，不公平，而需求真实存在。

## 功能

| 模块 | 说明 |
|------|------|
| 账号体系 | 注册 / 登录（服务端会话 Cookie，可随时吊销），角色 `admin` / `user`（Casbin 鉴权） |
| 凭据管理 | 平台凭据 AES-256-GCM 加密入库，仅任务下发瞬间解密；主密钥版本化可轮换；支持账号级粘性代理；每次解密留审计（用户可自查，可配置审计失败即拒绝使用）；推荐令牌/Cookie 优先 |
| 任务中心 | 按平台/动作创建任务，参数由脚本 manifest 动态表单；状态机与历史 |
| 实时进度 | WebSocket 事件流 + 进度存储 |
| 题库 | 答题结果沉淀，命中直接作答，减少 LLM 调用 |
| 脚本生态 | 平台脚本可插拔：GitHub 安装（zip/加速源/ref）、详情与实时 manifest、平台内调试、重新安装、启停与卸载（仅管理员） |

## 架构

```
浏览器 (Vue3 + Vuetify MD3)
        │ REST + WebSocket
        ▼
Go 后端 (Gin + GORM + 迁移 + Casbin) ── 编排 / 存储 / 鉴权 / 脚本安装
        │ gRPC AgentService（流式）
        ▼
Go Host (cmd/agent) ── 扫描脚本目录 / 生命周期 / 热扫描
        │ 子进程 + stdio JSON（任意语言）
        ▼
平台脚本（独立项目，可单独 CLI 运行）
        ▲
        └── gRPC QuestionBank（脚本经后端读写题库）
```

三层职责分离：前端只交互、后端只编排、Host 管脚本、脚本只执行（不直连数据库）。

## 运行模式

- **lite（默认）**：SQLite（纯 Go，无 CGO）+ 无 Redis，适合个人单机；`config.yaml` 首次运行自动生成。
- **full**：MySQL + Redis，适合多用户/服务器部署（Docker Compose）。

## 技术栈

- **前端**：Vue3 + JavaScript + Vite + Pinia + Vue Router + Axios + Vuetify（MD3）
- **后端**：Go + Gin + GORM + gRPC + golang-migrate + Casbin；lite 用 `github.com/glebarez/sqlite`
- **脚本**：任意语言（推荐 Python），通过 stdio JSON 协议接入

## 快速开始

### lite（单机，推荐先试）

```bash
# 1. 构建（前端可选）
cd frontend && npm install && npm run build && cd ..
cd backend
go build -o omnitask-server ./cmd/server
go build -o omnitask-agent  ./cmd/agent

# 2. 运行（首次生成 config.yaml：mode=lite，SQLite，随机密钥）
./omnitask-server -conf config.yaml
# lite 会自动拉起同目录的 omnitask-agent；也可 agent.autostart=false 后手动启动

# 3. 浏览器打开 http://127.0.0.1:8080 ，首个注册用户为管理员
# 4. 在「脚本」页从 GitHub 安装 cidaren / welearn
```

### full（推荐：Docker Compose 一键启动）

```bash
docker compose -f deploy/docker-compose/docker-compose.full.yml up -d --build
# 打开 http://<服务器IP>:4096/ ，首个注册用户为管理员
```

包含 MySQL、Redis、server、Host、nginx。配置与数据说明见
[deploy/docker-compose/README.md](deploy/docker-compose/README.md)。

只启动 MySQL/Redis 做本地开发：

```bash
docker compose -f backend/deploy/docker-compose/docker-compose.yml up -d
cd backend && go run ./cmd/server -migrate -conf config/deploy.yml
go run ./cmd/server -conf config/deploy.yml
go run ./cmd/agent  -conf config/deploy.yml
```

## 构建与工具

引导式工具箱（推荐）：

- Linux/macOS：`./omnitask.sh`
- Windows：`omnitask.bat`

运行后是数字菜单：部署、构建后端 server / Host agent（可选 **linux-amd64 / windows-amd64 / 两者**）、构建前端。产物输出到 `build/`（已 gitignore），命名如 `omnitask-server-linux-amd64`、`omnitask-server-windows-amd64.exe`。

部署：菜单选项「部署」或非交互 `./omnitask.sh deploy`（构建前端并内嵌 → 编译 → 迁移 → 重启）。

`Makefile` 仅为开发辅助（非构建必需）：`make proto` 需 `protoc` + `protoc-gen-go`/`protoc-gen-go-grpc` + `sdk/python/.venv`(grpc_tools)；`make backend-run` 需 Nunu CLI；`make sdk-test`/`frontend-dev` 分别需 Python venv / npm。

三个程序的职责、工作流程与构建参数详见 **[docs/build-and-run.md](docs/build-and-run.md)**。

## 脚本生态：开发与接入

平台能力全部来自**可插拔脚本**。脚本是独立项目，**不改文件名、不必改造既有项目**，三种接入方式任选：

- **包装仓**：已有项目零改动，加一个 `adapter.py` 映射 action。
- **装饰器**：`from omnitask_sdk import action, serve`，几行接入。
- **CLI 桥**：已有命令行工具，**零代码**，只写 `script.json`。

```bash
pip install "git+https://github.com/CometNetworkStudio/OmniTask.git#subdirectory=sdk/python"
```

官方脚本（可作模板）：[cidaren](https://github.com/CometNetworkStudio/cidaren) ｜ [welearn](https://github.com/CometNetworkStudio/welearn)

👉 **脚本开发指南（组件、manifest v2、本地调试）：[docs/script-development.md](docs/script-development.md)**
👉 **接入总览：[docs/integration-guide.md](docs/integration-guide.md)**
👉 **脚本协议：[docs/script-protocol.md](docs/script-protocol.md)** ｜ **SDK：[sdk/python/README.md](sdk/python/README.md)**

## 免责声明

本项目仅供个人学习与技术研究使用。请遵守各平台服务条款与学校相关规定，因使用本项目产生的一切后果由使用者自行承担。本项目不包含第三方平台脚本代码，脚本由作者独立发布并自负责任。

**凭据信任提示**：OmniTask 需要代用户登录目标平台，因此**任何部署实例在任务运行时都能在技术上解密用户凭据**。加密（AES-256-GCM）只能防止数据库/备份泄露，防不住恶意运维。请只使用你信任的实例，**不要使用来路不明的公开部署**；担心风险请自行本地部署。建议为各平台使用独立密码，避免与邮箱、银行等账号相同。

**已实施的基础加固**：

- 主密钥支持版本化与轮换（`security.master_keys` + `-rotate-credentials`），旧数据按记录的 `key_version` 解密；
- 会话使用服务端存储的随机会话，Cookie 为 `HttpOnly` + `SameSite=Lax`，数据库只存 SHA-256 摘要；禁用账号、吊销会话立即生效；
- 非安全方法要求 `X-CSRF-Token` 双提交校验，并校验 `Origin`；
- 请求日志只记录方法、路径、脱敏查询串与大小，响应日志只记录状态码/大小/耗时，**不记录任何请求体或响应体**（避免凭据明文与登录凭据入日志）；
- 下发 CSP、HSTS（HTTPS 时）、`X-Content-Type-Options`、`X-Frame-Options` 等安全响应头；
- 登录/注册按 IP 限流，并支持登录失败锁定；脚本子进程只继承白名单环境变量；
- `config.yaml` 在 Unix 上创建后收紧为 `0600`（Windows 请自行用文件 ACL 限制访问）；其中含主密钥，请勿提交或随数据库一起备份。

**残余风险**：恶意服务器仍可在任务执行时取得凭据明文；脚本以 Host 进程权限运行，安装来源不明的脚本等同于信任其代码。高敏感场景请自托管、只安装可信脚本，并限制实例访问范围。部署建议见 `docs/deployment-profiles.md`。

## License

[MIT](LICENSE) © CometNetworkStudio
