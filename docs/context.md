# OmniTask 万务通

> 一处托管，万务自达

面向大学生的**校园任务自动化聚合平台**。把重复、耗时的线上任务交给 Agent 自动完成，把时间还给学习本身。

> ⚠️ 本文件是项目的"上下文存档"。聊天记录可能丢失，**任何重要决策、约定、待办都必须回写到本文件**，不要只留在对话里。

---

## 1. 项目背景与目标

大学生在校期间要面对：英语四六级、计算机二级、普通话考试、学科竞赛，以及学校布置的各类线上任务（学习通视频时长、词达人、WeLearn 课程作业与挂时长等）。这些任务重复、耗时。

市面上有人掌握这些"信息差"（本质是 GitHub 上一些开源脚本）并借此牟利，而需求很大。本项目的目标是把这些能力**聚合到一个平台**：用户在前端一次配置，后端编排任务并下发指令，Agent（Python）自动执行，用户可实时查看进度，无需再开着浏览器页面、守着电脑或手机。

**项目定位：个人自用**（不做支付/多租户 SaaS，但架构上保留可扩展性）。

---

## 2. 已确认的关键决策（Decision Log）

| 项 | 决策 | 备注 |
|----|------|------|
| 项目名 | **OmniTask / 万务通** | |
| 产品形态 | **个人自用** | 不做支付、不做复杂多租户 |
| Agent 部署 | **服务器集中式** | 所有 Agent 跑在服务器，剥离 GUI，headless |
| 脚本来源 | **独立重写（未复制原码）** | 协议事实经所有者授权从参考源码提取，见 `docs/reference-analysis.md` |
| 开源自用 | **全部 MIT 开源** | 自己写、自己开源，杜绝原项目授权问题 |
| Go ↔ Python 通信 | **gRPC** | protobuf 强类型，支持 server-streaming 回传进度 |
| 前端 | Vue3 + **JavaScript（非 TS）** + Vite + Pinia + Vue Router + Axios | |
| 后端 | Go + Gin + MySQL + Redis + GORM/gen | 脚手架 Nunu |
| 首批平台 | **词达人、WeLearn** | 后续扩展学习通等 |
| 仓库布局 | **Nunu advanced 作 `backend/`**，根目录另建 `frontend/`、`sdk/python/`、`proto/` | 不采用 Nunu monorepo（其前端为 TS） |
| 前端脚手架 | **Vite `vue` 模板（纯 JS）** + Pinia + Vue Router + Axios | 忽略 Nunu 自带 web（TS） |
| 工具链版本 | Go 1.27.1 / Node 22.20.0 / Python 3.12 | Agent 用 `sdk/python/.venv` |
| proto 生成 | `make proto`（Go→`backend/api/agent/v1`，Py→`sdk/python/gen`），产物 gitignore | |
| 词达人凭据 | **仅接受 Token**（原项目无密码登录，靠 mitmproxy 抓取） | 见 `docs/reference-analysis.md` |
| 词达人答题 | **题库 → LLM → SkipAnswer**；`VerifyAnswer.answer_corrects` 权威回写题库 | 源码核对：`answer_tag` 是提交码，非正确标记（原设想已修正） |
| 开发期存储 | **Docker MySQL(3380) + Redis(6350)** | 复用 Nunu compose，避开系统端口 |
| 用户体系 | **注册/登录 + JWT**（少用户） | |
| 并发限流 | **全可配，默认低**（同账号串行，平台并发默认 2） | |
| 服务端口 | 后端 HTTP **8080**、Agent gRPC **50051**、前端 dev **5173** | 本机 8000 被其他容器占用 |
| 示例清理 | 删除 Nunu user 垂直示例、tests、mocks、占位 swagger json | 保留框架基础设施 |
| 前端 UI | **Vuetify 4.2.1**（`npm i vuetify` 最新版）+ `@mdi/font` + `vite-plugin-vuetify` | 官方 MCP 已接入 opencode |
| 答题 LLM | **DeepSeek**（OpenAI 兼容），key 走 env | 题库优先，省 token |
| 题库架构 | 后端 gRPC `QuestionBank` 服务 + Agent 客户端 | Agent 不直连 MySQL |
| WeLearn 风控 | 无图形验证码（已确认） | 直接账密登录 |

### 授权风险（重要，已核实）

- **Easy_Cidaren（词达人）**：仓库标注 GPL-3.0，但 README 写明"仅供个人学习、请勿商用、严禁盗卖盈利"。GPL-3.0 与作者声明冲突；GPL 有传染性，吸收其代码会导致本项目被迫 GPL。→ 因此**只参考其行为，不参考其代码，净室重写**。
- **Auto_WeLearn（WeLearn）**：**无 LICENSE 文件**（README 仅"免费"徽章），法律上默认保留所有权利，README 也禁止倒卖/代刷牟利。→ 同样**净室重写**。
- 结论：本项目代码独立实现，采用 **MIT** 协议对外（开源与发布由项目所有者手动执行）。

> **权衡说明（已拍板）**：M1 阶段缺少真实抓包样本，所有者授权阅读参考源码以**提取协议事实**（签名、jv 规则表、加密码算法、端点与参数顺序），再独立重写。实现代码为本项目原创、未复制原项目代码；但严格"净室"主张因此减弱，生产使用前建议以真实报文复核。

---

## 3. 技术栈

### 前端
- Vue 3（组合式 API）
- JavaScript（**明确不用 TypeScript**）
- Vite
- Pinia（状态管理）
- Vue Router
- Axios（HTTP）
- Vuetify 4 + MDI 图标（UI 组件库）

### 后端
- Go + Gin
- MySQL（持久化）
- Redis（缓存、任务队列/进度）
- GORM + **gorm/gen**（自动生成结构体）
- 脚手架：**Nunu**（https://github.com/go-nunu/nunu）

### Agent
- Python 3.12+
- gRPC（与 Go 通信）
- 剥离原脚本 GUI，改为可被调用的无界面服务/库

### 通信
- 前端 ↔ 后端：REST（Axios）+ WebSocket（进度实时推送）
- 后端 ↔ Agent：**gRPC**（定义见 `proto/`）

---

## 4. 三层架构

```
┌─────────────────────────────┐
│  前端 Vue3 (用户交互)        │  账号/Token 录入、任务选择、进度查看
└──────────────┬──────────────┘
               │ REST + WebSocket
┌──────────────▼──────────────┐
│  后端 Go + Gin (编排/存储)   │  用户数据、任务编排、指令下发、进度汇聚
└──────────────┬──────────────┘
               │ gRPC
┌──────────────▼──────────────┐
│  Agent Python (实际执行)     │  各平台脚本，headless，可并发
└─────────────────────────────┘
```

职责边界（分层分工，务必保持）：
- **前端只负责交互**，不直接碰任何平台。
- **后端不执行脚本**，只做用户数据、任务生命周期与指令下发/进度汇总。
- **Agent 只负责执行**，通过 gRPC 接收指令、回传进度，不直连数据库。

---

## 5. Monorepo 目录规划（已落地）

```
OmniTask/
├── README.md
├── AGENTS.md              # 项目规则（含前端 UI 规则）
├── Makefile               # make proto / sdk-test / backend-run / frontend-dev
├── omnitask.sh/.bat       # 引导式工具箱（部署 / 交叉编译）
├── .env.example           # 环境变量模板（.env 为本机实际值，gitignore）
├── proto/agent/v1/agent.proto   # gRPC 契约
├── backend/               # Go 后端（Nunu，module=backend）
│   ├── api/agent/v1/      # protoc 生成，gitignore，make proto 重建
│   ├── cmd/{server,agent}/        # 主服务 / 脚本 Host
│   ├── config/{local.yml, deploy.yml(gitignore), prod.yml(遗留)}
│   ├── internal/{agenthost,handler,middleware,model,repository,service,router,server,ws}/
│   ├── pkg/{config,crypto,authz,jwt,sid,log,server,...}/
│   └── deploy/docker-compose/   # MySQL/Redis（full）
├── frontend/              # Vue3 + Vuetify(MD3) + Pinia + Router + Axios
│   └── src/{router,api,views,stores,config,ws}/
├── sdk/python/            # 平台脚本 Python SDK
│   ├── core/              # http/ratelimit/bank/config/llm/params/events
│   ├── runtime.py         # stdio JSON 运行时（serve）
│   ├── gen/agent/v1/      # protoc 生成，gitignore
│   └── tests/
├── scripts/<id>/          # 已安装脚本（gitignore）
├── build/                 # 交叉编译产物（gitignore）
└── docs/
```

> 说明：平台脚本为**独立项目**（`CometNetworkStudio/{cidaren,welearn}`），由 Go Host 以子进程运行；`sdk/python/` 是其共享 SDK 源头。

---

## 6. 首批适配平台

| 平台 | 功能 | 原参考项目 | 授权 | 我们的做法 |
|------|------|-----------|------|-----------|
| 词达人 | 自动过学习任务、测试任务，得分 95+ | https://github.com/ularch/Easy_Cidaren | GPL-3.0 + 禁商用 | 净室重写 |
| WeLearn | 刷课程作业、刷学习时长、多账号 | https://github.com/jhl337/Auto_WeLearn | 无 LICENSE | 净室重写 |

后续规划（未开始）：学习通、知到等。

---

## 7. 核心数据流

1. 用户在前端登录 → 录入平台凭据（账号密码或 Token）。
2. 后端加密存储凭据，创建"任务"记录（平台、类型、参数、状态）。
3. 后端通过 gRPC 向 Agent 下发执行指令。
4. Agent 执行脚本，通过 gRPC server-streaming 持续回传进度/结果。
5. 后端将进度落 Redis + 汇总，经 WebSocket 推给前端。
6. 前端实时展示进度；完成/失败落库。

---

## 8. 待确认 / 待定问题

以下为默认取值，如有异议需修改并回写本文件：

- [ ] **进度推送**：暂定 WebSocket（前端），gRPC server-streaming（Agent→后端）。gRPC 侧已在 proto/Agent 落地，WebSocket 待做。
- [ ] **凭据存储**：暂定 AES-256-GCM 加密入库，密钥走环境变量。
- [x] **原脚本 GUI**：全部剥离，改为 headless 服务/库。
- [x] **Nunu layout**：采用 **advanced**（`backend/`，预留 job/task 层；其中 `web/` 占位已删）。
- [ ] **gorm/gen 约定**：模型生成到 `internal/model`，query 生成到 `internal/repository/query`。
- [x] **部署**：暂定 Docker Compose（mysql / redis / backend / agent / nginx）。
- [x] **认证方式**：注册/登录 + JWT（个人自用，少用户）。
- [x] **预期并发量**：全可配，默认低（同账号串行、平台并发默认 2）。
- [ ] **题库归属**：后端持有 MySQL 题库，Agent 不直连库（详见 `docs/development-plan.md`）。

---

## 9. 里程碑（Roadmap）

> 详细任务、验收与风险见 **`docs/development-plan.md`**。开发顺序：契约 → 平台协议(Python) → 后端 → 通讯 → 前端 → 集成 → 部署。

- [x] M0：monorepo 骨架、Nunu 后端（`backend/`，8080）、Vue 前端（`frontend/`）、proto 契约、Python Agent gRPC 骨架（冒烟通过）
- [x] M0.5：契约补全——proto `QuestionBank.Lookup/Save`、`internal/model/model.go` 冻结 5 表 schema、`.env.example` 环境变量约定（`make proto` + Go build/vet + Python import 均通过）
- [x] M1：平台协议层（Agent）——词达人、WeLearn 无界面实现 + `run_cli.py`；单元测试通过。**已真机联调**：词达人完成 List 04（196/196，题库回写 129 条，含人类思考延迟）；WeLearn 完成课程 2416 全部 144 个 SCO（挂时长 + 提交进度两种模式）
- [x] M2：后端领域层——User/Credential/Task/Question repository + service（bcrypt+JWT、AES-256-GCM 凭据、任务/题库）+ REST（`/api/v1/auth|credentials|tasks|questions`）+ GORM 迁移（MySQL 3380 / Redis 6350）；curl 验收通过，凭据确认为密文
- [x] M3：通讯层——Go gRPC 客户端（Agent 50051）+ `QuestionBank` gRPC 服务端（9000）+ 任务编排（pending→queued→running→终态、凭据解密、Redis 进度、TaskEvent）+ WebSocket `/api/v1/ws`；端到端联调通过（WebSocket 实时事件、成功/失败终态、题库 gRPC 读写）
- [x] M4：前端——登录/注册、凭据管理、任务列表+参数表单创建、任务详情（WebSocket 实时进度）、题库；Vuetify 4 + 主题 + Pinia + api/service 层，严格遵循前端 UI 规则（`npm run build` 通过，Vite 代理联调通过）
- [ ] M5：集成测试与加固
- [ ] M6：Docker 化部署与文档

### M0 验收命令

```bash
make proto          # 生成 Go/Python proto 代码（产物已 gitignore）
make sdk-test     # Python Agent gRPC 冒烟测试
cd backend && go build ./... && go vet ./...
cd frontend && npm run build
```

---

## 10. 开发约定

- **语言**：前端 JavaScript（禁止引入 TS）；后端 Go；Agent Python。
- **不抄原脚本代码**：只依据平台网络接口/协议自主实现，保留独立实现证据（见 `docs/reference-analysis.md`）。
- **密钥/凭据永不入库明文、永不提交到仓库**。
- **无注释除非必要**；命名与现有代码风格保持一致。
- 每次重要改动后更新本 README 的 Decision Log 与 Roadmap。
- 提交前运行：`make sdk-test`、`cd backend && go build ./...`、`cd frontend && npm run build`；改 proto 后先 `make proto`。
- proto 生成产物（`backend/api/agent/`、`sdk/python/gen/`）不入库，由 `make proto` 重建。
- Agent 依赖装在 `sdk/python/.venv`，不污染系统 Python。

---

## 11. 免责声明

本项目仅供个人学习与技术研究使用。使用者需自行遵守各平台服务条款与学校相关规定，因使用本项目产生的一切后果由使用者自行承担。

---

## 12. 后续决策（M5+）

- **角色/权限**：仅 `admin` / `user` 两角色；后端引入 **Casbin**（RESTful model，`keyMatch2`），按 `(role, route, method)` 鉴权；**首个注册用户自动为 admin**；「脚本」页仅 admin 可见。前端同一套，按角色条件渲染。
- **数据库迁移**：full 用 **golang-migrate**（`internal/migration`，embed SQL；`server -migrate` 或 `data.db.migrate_on_start`）；lite 用 AutoMigrate。
- **README**：根目录改为面向 GitHub 的展示型 README；原上下文存档移至 `docs/context.md`；接入文档 `docs/integration-guide.md`。
- **脚本插件化方向**：`manifest(script.json)` + gRPC 契约（`AgentService.Execute` 已是通用）；仅**管理员**安装、固定 commit、安装前审核；脚本独立仓库、**不入主库并 gitignore**；恶意脚本风险由使用者自辨，平台声明免责。
- **前端规则**：MD3 圆角（`elevated`+`rounded=xl`，card `elevation=2`）、含输入对话框 `persistent`、统一 `v-snackbar-queue`（见 `AGENTS.md`）。
- **部署**：`./omnitask.sh deploy` 一键构建+迁移+重启；公网 `https://<your-host>:4096/omnitask/`；nginx 497 回退到 HTTPS。

## 13. 脚本插件化决策（2026-09）

- **采用方案 C**：Go Host（`cmd/agent`，可与 backend 同构）+ 子进程脚本 + **stdio NDJSON** 协议；脚本任意语言，可独立单跑。
- 运行时缺失（如未装 Python）→ Host 预检标记不可用并提示；接受“检测并提示用户装 Python”。
- 脚本 manifest `script.json` 含 `runtime/entry/requires/actions/params/credentials`；规范见 `docs/script-protocol.md`。
- 安装：后端接收 GitHub 链接 → 下载 zip 到 `scripts/<id>/`（预留加速源，前端拼接）；仅管理员；固定 commit。
- 多机 Agent：**暂不做**（未来再加 `agent_id`/心跳）。
- full/lite 双模式：待脚本化稳定后做；lite 用 `github.com/glebarez/sqlite`（纯 Go）+ 无 Redis；密钥明文入 `config.yaml`（拍板）。

## 14. 脚本生态落地（2026-09）

- 独立脚本仓库已发布并开源（MIT + 净室重写声明）：
  `github.com/CometNetworkStudio/cidaren`、`github.com/CometNetworkStudio/welearn`。
- OmniTask 新增「脚本」页（仅 admin）与注册表：`scripts` 表（迁移 000002）；后端接收 GitHub 链接/zip，下载解压到运行目录 `scripts/<id>/`（支持加速前缀，如 `https://gh-proxy.com/`）；Host **热扫描**，安装后无需重启。
- 运行目录改为仓库根 `scripts/`（**gitignore**）；内置脚本已从主库移除，改为“安装”。
- `task` 创建校验：内置集合 ∪ 已启用脚本。
- 建议：加速域名用户写了 `.com` 与示例 `.org` 两个，前端都提供，按可用者选。

## 15. lite/full 双模式（2026-09）

- **统一配置 `config.yaml`**：`mode: lite|full`，默认 **lite**；仅当使用默认文件名 `config.yaml` 且不存在时自动生成（含随机 `jwt.key` / `master_key`，明文）。
- **lite**：`github.com/glebarez/sqlite`（纯 Go 无 CGO）+ **无 Redis**（进度走进程内 `ProgressStore`）；启动时 GORM AutoMigrate。
- **full**：MySQL + Redis；迁移用 golang-migrate（`server -migrate`）。
- 分层切换：业务代码零改动，仅 `repository.NewDB`（驱动）与 `NewRedis`（是否连接）按 `mode` 分支。
- 说明：golang-migrate 的 sqlite 驱动依赖 CGO，与“纯 Go”目标冲突，故 lite 采用 AutoMigrate；接受该取舍。
- ✅ `go:embed` 前端 dist：后端单文件托管 SPA（`internal/server/static.go`），HTTP API 统一前缀 `/api/v1`，`/api/*` 未命中返回 404，其余走 SPA 回退。

## 16. 引导式构建脚本（2026-09）
- 新增 `omnitask.sh`（Linux/macOS）与 `omnitask.bat`（Windows）引导式菜单：部署 / 编译 server·agent（可选 linux-amd64、windows-amd64、两者）/ 构建前端。
- 交叉编译：`CGO_ENABLED=0 GOOS/GOARCH=amd64`（依赖全为纯 Go：glebarez/sqlite、mysql、casbin）；产物在 `build/`（gitignore）。
- 部署逻辑已并入 `omnitask.sh`（非交互 `./omnitask.sh deploy`）与 `omnitask.bat`；`deploy.sh` 已删除。
- Makefile 仅为开发辅助（proto 生成/nunu/测试），普通构建不需要。

## 17. 目录更名（2026-09）
- 根目录 `agent/` → `sdk/python/`：定位为「Python 脚本 SDK 源头（core/runtime/gen/tests）」。
- 删除历史遗留 `grpc_server.py`（旧 Python gRPC Agent）与 `test_smoke.py`；Host 已由 Go 实现（`backend/cmd/agent` + `backend/internal/agenthost`）。
- `Makefile`：`proto-py` 输出到 `sdk/python/gen`；`agent-test` 更名为 `sdk-test`。
- 配置：Host `agent.pythonpath` 默认 `sdk/python`；`local.yml`/`deploy.yml` 同步。
- 新增 `sdk/python/README.md`（SDK 用法与接入）。

## 18. 迁移并入 server + HTTPS 配置（2026-09）
- **迁移（方案A）**：删除 `cmd/migrate`，迁移逻辑移入 `internal/migration`（embed SQL）；由 `omnitask-server -migrate` 触发，或配置 `data.db.migrate_on_start: true` 启动时执行。产物由 3 个减为 **2 个**（server + agent）。
- **HTTPS**：`config.yaml` 新增 `http.tls.{enabled,cert_file,key_file}`，server 可自行终结 TLS（不做自签）。
- **配置清理**：删除未使用的 `backend/config/prod.yml` 与过期的 `frontend/.env.development`。
