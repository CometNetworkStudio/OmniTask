# OmniTask 万务通

> 一处托管，万务自达

面向大学生的**校园任务自动化聚合平台**。把重复、耗时的线上任务交给 Agent 自动完成，把时间还给学习本身。

> ⚠️ 本文件是项目的"上下文存档"。聊天记录可能丢失，**任何重要决策、约定、待办都必须回写到本文件**，不要只留在对话里。

---

## 1. 项目背景与目标

大学生在校期间要面对：英语四六级、计算机二级、普通话考试、学科竞赛，以及学校布置的各类线上任务（学习通视频时长、词达人、WeLearn 课程作业与挂时长等）。这些任务重复、耗时。

市面上有人掌握这些"信息差"（本质是 GitHub 上一些开源脚本）并借此牟利，而需求很大。本项目的目标是把这些能力**聚合到一个平台**：用户在前端一次配置，后端编排任务并下发指令，Agent（Python）自动执行，用户可实时查看进度，无需再开着浏览器页面、守着电脑或手机。

**项目定位：官方免费实例 + 自托管**（面向学生免费开放，不做支付；架构上保留可扩展性）。

---

## 2. 已确认的关键决策（Decision Log）

| 项 | 决策 | 备注 |
|----|------|------|
| 项目名 | **OmniTask / 万务通** | |
| 产品形态 | **官方免费实例 + 自托管** | 官方实例免费开放；不做支付、不做复杂多租户 |
| 官方实例 | **app.omnitasks.top** | 主域 `omnitasks.top` 为静态门户；用户注册即用 |
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
- [x] M5（部分完成）：已有 SDK/加密单测与多次真机联调；**缺**后端 service/handler 测试、CI、E2E 自动化、并发压测
- [x] M6：部署与文档（单文件内嵌前端、Docker Compose、nginx 反代、lite/full、配置自动生成、TLS 可选、引导脚本）

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
- **脚本插件化方向**：`manifest(omnitask.json)` + gRPC 契约（`AgentService.Execute` 已是通用）；仅**管理员**安装、固定 commit、安装前审核；脚本独立仓库、**不入主库并 gitignore**；恶意脚本风险由使用者自辨，平台声明免责。
- **前端规则**：MD3 圆角（`elevated`+`rounded=xl`，card `elevation=2`）、含输入对话框 `persistent`、统一 `v-snackbar-queue`（见 `AGENTS.md`）。
- **部署**：`./omnitask.sh deploy` 一键构建+迁移+重启；公网 `https://<your-host>:4096/omnitask/`；nginx 497 回退到 HTTPS。

## 13. 脚本插件化决策（2026-09）

- **采用方案 C**：Go Host（`cmd/agent`，可与 backend 同构）+ 子进程脚本 + **stdio NDJSON** 协议；脚本任意语言，可独立单跑。
- 运行时缺失（如未装 Python）→ Host 预检标记不可用并提示；接受“检测并提示用户装 Python”。
- 脚本 manifest `omnitask.json` 含 `runtime/entry/requires/actions/params/credentials`；规范见 `docs/script-protocol.md`。
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

## 19. 脚本生态：SDK 正式化 + 三种接入（2026-09）
- 不强制文件名：manifest 支持 `entry`（文件）或 `module`（`python -m`），二者均任意。
- Python SDK 正式化：`sdk/python` 包名 `omnitask_sdk`，`pyproject.toml` 可 `pip install git+...#subdirectory=sdk/python`；protobuf 生成物随包提交（`omnitask_sdk/gen`）。
- 三种接入：A 包装仓（`adapter.py`，原项目零改）、B 装饰器 `@action`+`serve`、C CLI 桥（仅 `omnitask.json`，`module: omnitask_sdk.bridge` + `actions[].command` + `progress_regex`）。
- 新增 `AdapterContext`（params/credentials/progress/log/result）。
- Host：支持 manifest `module`；SDK 运行时支持生成器与装饰器两种。
- Makefile `proto-py` 输出改到 `sdk/python/omnitask_sdk/gen` 并修正生成文件导入。
- 文档重写：integration-guide（三范式+走查+Vibecoding Prompt）、script-protocol（module/command/progress_regex）、sdk README。

## 20. 进度盘点与待办（2026-09）

### 里程碑现状
- M0–M4 ✅；M6 部署 ✅（含单文件/nginx/Docker/lite-full/TLS）。
- M5 集成与加固 **部分**：有 SDK(17)/加密单测与多轮真机联调；缺后端 service/handler 测试、CI、E2E 自动化。
- 原里程碑之外已完成：脚本插件化（安装/注册表/Go Host/stdio 协议/SDK）、角色权限(Casbin)、golang-migrate 并并入 server、API 前缀 `/api/v1`、SDK 生态（omnitask_sdk + 装饰器 + CLI 桥）。

### 未解决 / 待办（优先级）
**P0 稳固**
- 任务生命周期：无取消/删除/重试接口；server 重启后 `running` 任务成僵尸；无超时。
- 注册控制（开放注册）、生产禁用 Swagger、收紧 CORS。
- 基础 CI：`go test ./...` + `npm run build` + SDK 测试一条命令。

**P1 体验/功能**
- `/api/v1/me`（前端启动刷新角色）；用户管理（admin 改角色/禁用）。
- 任务列表分页/筛选；脚本安装进度、指定 commit/tag、运行时缺失指引。
- backend/agent systemd 化（服务器重启自启动）。

**P2 生态/长期**
- 脚本依赖自动安装（隔离 venv）或明确提示；SDK 多语言示例；新平台（学习通等）。
- 多机 Agent（已评估，暂不做）。

### 工程卫生
- 已删除 Nunu 遗留 `backend/cmd/{task,migration}`、`backend/scripts/`；`backend/Makefile` 精简。
- lite(AutoMigrate) 与 full(SQL 迁移) 双机制，注意 schema 漂移。
- cidaren 11003 风控（运营项，需人工安全验证）。

## 21. P0 加固完成（2026-09）
- **产品方向：官方免费实例（小范围多用户起步）**。P0 已完成，剩余项见 `docs/todo.md`（完成后可清空）。
- 任务生命周期：`POST /api/v1/tasks/:id/cancel`、`/retry`、`DELETE /api/v1/tasks/:id`；编排器持取消句柄，取消时中断 Host 子进程；启动把 `running/queued` 置 `interrupted`；`agent.task_timeout` 可配超时。
- 注册控制 `security.registration: open|closed`；生产默认关闭 Swagger（`http.swagger`）；CORS 白名单 `http.cors_origins`（默认同源）。
- 自检：`./omnitask.sh check`（后端 build/vet/test + 前端 build + SDK 测试）；GitHub Actions `.github/workflows/ci.yml`。
- 前端：任务详情与列表支持取消/重试/删除。

## 22. P1（部分）完成（2026-09）
- `/api/v1/me`；用户管理（admin）：`GET /users`、改角色、启用/禁用；前端「用户」页（admin 路由）。
- 分页：`pagination.default|max` 配置；前端 `components/DataTable.vue` 统一封装（MD3 风格 + 自研分页页脚）。
- 布局：抽屉分组导航 + append 底部账号/退出/版权；顶栏头像用 `v-btn` 包装 + `me-2` 修复裁剪；移动端临时抽屉。全程无自定义 CSS（合规）。
- 用户禁用：登录返回 `1013 账号已被禁用`。
- 剩余 P1（筛选、脚本页体验、systemd）见 `docs/todo.md`。

## 23. P1 收尾（2026-09）
- 任务列表筛选：`GET /tasks?status=&platform=`（后端查询参数 + 前端下拉）。
- 脚本安装支持 `ref`（分支/标签/commit，commit 走 codeload `/zip/<sha>`）；「运行时缺失」帮助弹窗。
- **systemd 服务化**：`deploy/systemd/{omnitask-server,omnitask-agent}.service` + `install.sh`；已在本机 enable+active，`omnitask.sh deploy` 检测到 unit 后改用 `systemctl restart`。
- 产物与日志目录改为稳定路径 `bin/`、`logs/`（gitignore；不再用 /tmp，避免重启丢失）。
- 抽屉仿 `AppLayout.vue`：去掉顶部 hero、分组导航、底部 append（主题/退出）、版权；`v-list-item` 加 `rounded="xl"`；允许必要 scoped CSS（滚动/隐藏滚动条/spacer/禁选）。
- 剩余（见 `docs/todo.md`）：真实安装进度、脚本依赖自动安装。
- 说明：ID 维持现状（Sonyflake），不做 UUIDv7 改动。

## 24. 清理与依赖自动安装（2026-09）
- 删除 Nunu 遗留：`backend/cmd/task`、`backend/cmd/migration`、`backend/scripts/`；精简 `backend/Makefile`。
- 补测试：`internal/service`（pagination/user，in-memory SQLite）、`pkg/authz`（Casbin RBAC）。
- 依赖自动安装：安装脚本时若含 `requirements.txt` 且有 `agent.auto_install_deps: true`，在脚本目录建独立 `.venv` 并 pip 安装（`agent.pip_index_url` 可配加速源）；Host **优先使用脚本自带 `.venv` 的解释器**。
- 2026-11 调整：默认开启 `auto_install_deps`（部署/本地配置与默认模板），安装进度显示「安装依赖」阶段；部署配置的 pip 源指向清华镜像。
- 脚本页安装增加 indeterminate 进度条与步骤文案；安装请求超时放宽到 10 分钟。
- 抽屉宽度改回 Vuetify 默认（移除 `width="280"`）。

## 25. 脚本安装真实进度（2026-09）
- 安装改为**异步任务**：`POST /scripts/install` 立即返回 `job_id`；`GET /scripts/install/:job` 轮询进度。
- 进度来源真实：下载按字节百分比（1–59%）→ 解压 60% → 写入 68% → 依赖安装（流式读取 pip 输出，74–97%）→ 注册 98% → 完成 100%。
- 安装任务内存态（单实例），15 分钟超时；失败返回 `error`。
- 前端：脚本页显示 determinate 进度条 + 阶段文案，轮询 700ms；安装请求超时 30s（后续轮询），整体最长 15 分钟。
- 验证：真机 job `download 1% → done 100%`，安装成功。

## 26. 接入学习通（chaoxing，GPL-3.0，隔离）(2026-09)
- 选用**方案 A（GPL 隔离）**：`CometNetworkStudio/chaoxing`（fork 自 `Samueli924/chaoxing`，GPL-3.0）作为**独立脚本仓库**，不并入 MIT 主库；通过进程边界接入。
- 该 fork 新增：`omnitask_entry.py`（stdio 适配器，生成临时 config.ini 调 main.py、合并 stderr、解析 `NN%` 进度）、`omnitask.json`（manifest：动作 study/learning_count、凭据账号密码、参数 course_list/speed/jobs/…）；`requirements.txt` 追加 `omnitask-sdk`。已推送。
- 关键适配点：该工具日志/进度走 **stderr**（loguru+tqdm）→ 适配器 `stderr=STDOUT` 捕获；`course_list` 为空会交互式 `input()` → 适配器传 `all` 规避。
- 验证：OmniTask 脚本页安装成功；Host `Health` 识别 `chaoxing`；任务链路 后端→Host→适配器→main.py 正常（因未装 chaoxing 依赖按预期 `failed`）。
- 实机运行需：`agent.auto_install_deps: true`（安装其较重依赖，含 ddddocr）或手动装；建议 Python 3.13。
- 许可：该仓库整体仍 GPL-3.0；上游“禁止盈利/闭源商用”声明继续有效；本项目/its SDK 保持 MIT，仅进程边界协作。


## 27. 平台发现（Discovery）与平台 schema 上后端（2026-09）
- **proto**：`AgentService.Discover(DiscoverRequest) → DiscoverResponse{items[key,fields]}`；SDK 新增 `@listing()`；Host 新增 `Discover`（子进程发送 `type:discover`，收集 `type:items`）。
- **后端**：`platform_caches` 表（迁移 000004，items 存 JSON、刷新人限流 `discovery.min_interval` 默认 60s）；`DiscoverService`；接口 `GET /api/v1/platforms`（完整 manifest schema，任意登录用户）、`GET /api/v1/platforms/:id/items`、`POST /api/v1/platforms/:id/discover`；Casbin 放行 user。
- **脚本**：cidaren/welearn/chaoxing 均新增 `@listing` 与 manifest `discovery`（已推送三个脚本仓库）。
- **说明**：注册表里的 manifest 是**安装时**快照，新增 discovery 后需**重新安装**脚本以刷新；已在部署环境重装并验证 `/platforms` 返回三平台 schema。
- **待办**：前端改为从 `/platforms` 动态获取（删除写死 platforms.js）、发现列表 UI、Home/引导/名句。

## 28. 脚本契约 v2：组件化与可视化（2026-11）
- **目标**：外部脚本低门槛接入，脚本用平台组件描述执行过程，前端通用渲染；不为旧脚本兼容做妥协（开发期）。
- **manifest v2**：`protocol: 2`；`actions[].description/params`；字段支持 `required/placeholder/help/options/min/max/step/multiline`；安装与加载校验 id、runtime、入口、action/字段唯一性，v2 至少一个 action。
- **事件 v2**：`started/progress/step/log/metric/succeeded/failed`；`progress` 支持 `current/total/unit/stage`；新增 `has_percent`，非进度事件不改变任务进度；Host 对非协议 stdout 输出给出一次警告。
- **SDK 组件**：`ctx.progress`/`ctx.progress_of`/`ctx.step`/`ctx.log`（debug/warn/error）/`ctx.metric`/`ctx.result`；CLI 桥支持 `[progress]`、`[step]`、`[metric]`、`[log]` 标记。
- **开发者工具**：`python -m omnitask_sdk.cli check|run|schema`，安装后 `omnitask-script`；发布 `script.schema.json`。
- **可视化**：`task_events` 增加 kind/level/step/status/current/total/unit/stage/key/value（迁移 000007）；新增 `GET /api/v1/tasks/:id/events`；任务详情展示步骤、指标、结果、日志与历史事件回放。
- **后端校验**：任务创建时按 manifest 校验 action 与必填参数；失败保留最后进度。
- **文档**：新增 `docs/script-development.md`，同步协议、接入指南、SDK README。

## 29. 脚本管理闭环（2026-11）
- **Host 实时清单**：proto 新增 `ListScripts`，返回完整 manifest、sha256、运行时路径与可用性；后端 `ScriptService.Sync` 以 Host 为准刷新注册表，改 `omnitask.json` 无需重装，Host 不可达时不误标缺失。
- **脚本详情**：`GET /api/v1/scripts/:id` 返回 manifest、动作、来源、ref、commit、hash、状态、最近错误；前端新增 `/scripts/:id` 详情页。
- **平台内调试**：`POST /api/v1/scripts/:id/debug` 直接运行脚本并返回组件事件，仅管理员；前端详情页提供参数、凭据与事件面板。
- **安装管理**：安装任务落库 `script_install_jobs`（迁移 000008），重启可查；`scripts` 增加 `ref/commit_sha/manifest_hash/last_error`；新增 `POST /scripts/:id/reinstall` 与 `POST /scripts/:id/rollback`，安装前保留上一版本，前端支持「重新安装 / 回滚」。
- **结果 schema**：`actions[].result` 支持 `table/links/keyvalue`；`ExecuteEvent.result_json` 保留数组/对象结构，`TaskResponse.result` 改为任意 JSON。
- **发现筛选**：`POST /platforms/:id/discover` 接受 `params`，带筛选查询不读写缓存；平台页新增筛选、批量勾选与批量执行。
- **并发与取消**：后端实现全局/平台/账号三级并发（`agent.global_concurrency`、`agent.platform_concurrency`）；Host 取消先发 `{"type":"cancel"}`，3 秒后强杀；SDK 提供 `ctx.canceled/ctx.check_canceled()`。

## 30. 部署与出口代理（2026-11）
- **Full 一键 Compose**：`deploy/docker-compose/docker-compose.full.yml` 启动 MySQL、Redis、server、Host、nginx；多阶段 Dockerfile 内构建前端 + proto + Go 产物；配置模板首次复制到 Docker 卷，避免把随机密钥写回仓库；脚本与日志共享 `omnitask-data` 卷。
- **Lite 单命令**：`agent.autostart` 默认在 lite 为 true，server 启动时检测端口并拉起同目录 `omnitask-agent`；可用 `agent.bin` 指定路径，或设为 false 手动启动。
- **账号级粘性代理**：凭据可配置 `proxy`，加密存储并生成脱敏 `proxy_hint`；任务/发现解密时提取代理，经 gRPC `proxy` 字段由 Host 注入 `OMNITASK_PROXY`；SDK `new_session()` 自动读取该环境变量。调试接口也支持临时 proxy。
- **多机决策**：当前仍是单 Host；多机需要节点注册、心跳、脚本分发、账号亲和与 TLS，暂不做。IP 风控优先用账号级代理，而不是直接拆多机。

## 31. 并发隔离与版本化脚本（2026-11）
- **每任务工作区**：Host 将脚本源码复制到 `<workspaces_dir>/task-<id>/`（排除 `.venv/.git/node_modules/缓存`），以工作区为 cwd，注入 `OMNITASK_WORKDIR`、`OMNITASK_SCRIPT_DIR`、`TMPDIR`、`HOME`、`XDG_*`；`config.ini`、Cookie、缓存等不再跨任务/账号互相覆盖。失败工作区默认保留，便于排查。
- **版本化脚本目录**：安装写入 `scripts/<id>/versions/<hash-time>/`，`active.json` 记录 active/previous；回滚只切换指针；旧版本按 `agent.keep_versions` 和 24 小时保护期清理。运行中的任务使用工作区副本，不受更新影响。
- **Host 并发保护**：新增 `agent.max_processes`（默认 8）与 Host 级平台并发；任务、发现、调试统一受进程额度约束。
- **队列与用户配额**：`task.max_active_per_user`（默认 20）、`task.max_pending`（默认 500）；创建任务时拦截，避免单用户打满队列。
- **跨进程节流**：Host 注入 `OMNITASK_RATE_DB`，SDK `RateLimiter` 用 SQLite（WAL + BEGIN IMMEDIATE）协调同一 Host 上多个脚本进程的平台请求间隔；未使用 SDK 限速器的脚本仍按进程独立。
- **配置**：新增 `agent.workspaces_dir/rate_dir/max_processes/keep_workspaces/keep_failed_workspaces/keep_versions`；Docker full 与 local.yml 同步。

## 32. 凭据安全加固（2026-11）
- **决策**：评估「浏览器本地存凭据」与「账号密码派生密钥」后，拍板维持服务端 AES 加密 + 无人值守执行，只做加固（方案 A）。浏览器本地存储防不住恶意服务端/XSS，且会破坏定时与无人值守任务；账号密码派生密钥只防备份泄露，仍防不住运行时代解密，且带来改密/找回/多端解锁的复杂度。
- **修复日志泄露**：原 `RequestLogMiddleware` 记录完整请求体（凭据明文）、全部请求头（含 Authorization）与完整 URL（WebSocket accessToken），`ResponseLogMiddleware` 记录响应体（登录 Token）。现改为仅记录方法、路径、脱敏查询串、请求大小，以及状态码/响应大小/耗时。
- **主密钥版本化**：新增 `crypto.Keyring`、`credentials.key_version`（迁移 000010）与 `security.master_keys`；映射非空时只读映射，新写入用最大版本，旧数据按记录版本解密。新增 `-rotate-credentials`：批量重加密到当前版本，成功后再移除旧密钥。`config.yaml` 在 Unix 上创建后 `chmod 0600`（Windows 需自行设置 ACL）。生成密钥时确保含字母，避免 YAML 把纯数字串当数值回写。
- **审计与展示**：任务/发现/编辑解密均写访问日志，并更新 `last_used_at`；`security.credential_audit_fail_closed=true` 时审计失败即拒绝解密。凭据页新增「最后使用」列。
- **传输与前端**：HTTPS 时下发 HSTS；全局 CSP（`script-src 'self'`，样式允许内联）、`X-Frame-Options`、`X-Content-Type-Options`、`Referrer-Policy`；API 响应 `Cache-Control: no-store`；登录/注册按 IP 限流（`security.auth_rate_limit_per_minute`，默认 20）。
- **残余风险**：恶意部署者仍可在运行时取得凭据明文；JWT 仍在 `localStorage`，XSS 可窃取会话；Agent gRPC 仍为明文（默认仅监听回环，远程部署需私网/VPN 或后续补 TLS）。

## 33. 平台页头部与批量操作（2026-11）
- **头部压缩**：平台页顶栏改为单行 flex，不再用 `flex-md-nowrap` 的网格列；筛选、表格/卡片切换、刷新均为图标按钮（带 tooltip），顺序为筛选、视图切换、刷新。搜索框和账号下拉恢复原来的 outlined 样式；桌面端搜索框靠右，紧邻账号下拉，宽度 200px（账号下拉 160px）。
- **响应式**：`md` 起显示截断的平台名，`lg` 起显示版本；刷新时间改到刷新按钮的 tooltip（`刷新于 xx`）。`xs` 隐藏平台标识，账号选择改为图标菜单；图标按钮保持默认尺寸不缩小，只压缩搜索框（小屏最小 48px）。表格卡片加 `overflow-x-auto`，窄屏横向滚动不再撑宽整个页面导致右侧按钮被挤出卡片。
- **Snackbar**：全局 `v-snackbar-queue` 使用 `rounded="xl"`（24px），与全局 MD3 圆角一致。
- **批量操作**：移除选中后插入的批量卡片，改为右下角常驻位置出现的 `v-speed-dial`；FAB 显示已选数量，展开后按动作直接打开批量执行弹窗（预选该动作），另提供清除选择。speed dial 的所有子项必须带 key，否则 Vuetify 的 TransitionGroup 会告警。卡片视图补上每张卡的勾选框，两种视图都可多选。

## 34. 页面细节与文案（2026-11）
- **任务详情间距**：内容区 `v-row` 改为 `density="comfortable"`，并把信息卡到步骤/指标区改为 `mt-4`，步骤/指标、结果、执行记录之间统一为 16px。
- **凭据高级项**：去掉代理字段前的 `v-divider` 和长说明，改为标题为「高级」的 `v-expansion-panels`；编辑已有代理时在面板标题显示脱敏提示，清除代理开关放在面板内。
- **脚本详情**：基本信息改为 8 个字段的 2 行 4 列（补全缺失的 `md="3"`），内容区加 `pa-5` 留白；来源只显示 `x-small` 的 GitHub 图标按钮；动作改为 `v-list`，描述和参数 chips 放在 `v-list-item` 默认内容区，避免 `v-list-item-subtitle` 的行高裁剪；Manifest 收进手风琴；回滚/卸载收进右上角更多菜单。
- **名句**：首页名句扩充到 35 条；`工欲善其事，必先利其器。` 从 `quotes.js` 移到引导步骤条第一步的描述上方。

## 35. 脚本输出编码与 WeLearn 行为对齐（2026-11）
- **乱码根因**：Windows 下 Python 的 stdin/stdout 走管道时默认本地编码（cp936），Host 按 UTF-8 解析 JSON，中文被替换为 U+FFFD；另有一批历史 manifest 是 UTF-8 被按 GBK 解码后入库，标签显示为乱码。
- **编码修复**：Host `scriptEnv` 注入 `PYTHONUTF8=1`、`PYTHONIOENCODING=utf-8`；SDK `serve` 启动时强制 stdin/stdout/stderr 为 UTF-8，`_emit` 直接以 UTF-8 字节写 stdout；legacy 生成器把最后一个非空 `result` 作为任务结果返回。
- **WeLearn 对齐原项目**：`time` 处理全部可见 SCO（不再只刷未完成，已完成课程也能挂时长）；`progress` 默认只提交未完成 SCO，`include_completed=true` 才重提已完成项；按 SCO 与单元隔离异常并继续；分别校验 `setscoinfo`/`savescoinfo` 的 `ret:0`；SCO id 优先 `id`；日志使用 `location` 课程名；最终事件返回 total/succeeded/failed/units_failed，单项进度封顶 99。
- **清单修复**：welearn 升到 0.1.1，`Sync` 按新 hash 覆盖数据库中的乱码 manifest；历史任务日志中的 U+FFFD 为有损替换，未回填。
- **真实测试**（凭证 `welearn/zym`）：发现 4 门课程；`time`（课程 2416，limit=1，3 秒）在已完成课程上成功处理 1 个 SCO；`progress`（课程 566，`include_completed=true`，limit=1）成功提交 1 个 SCO；DevTools 事件中文、结果 `{total,succeeded,failed}` 与 demo 脚本均正常。

## 36. 平台发现丰富与执行对话框整理（2026-11）
- **WeLearn 脚本**：独立仓库 `CometNetworkStudio/welearn` 已提交并推送 3 个 commit（挂时长/提交校验修复、发现进度与状态、按动作拆分参数），版本 0.1.3；后续 0.1.4 见下节。README 明确脚本与 OmniTask 主项目相互独立。
- **发现字段**：WeLearn 课程接口可用的展示字段为 `per`（进度）与 `state`（状态，按 per 派生）；columns 改为 课程名/课程ID/进度/状态，`primary=name`。卡片标题使用课程名，右上角用 `v-progress-circular` 显示进度环；表格进度列用 `v-progress-linear` 并设最小列宽。
- **响应式视图**：平台页按 `smAndDown` 区分小屏/大屏，分别用 `omnitask_platform_view:small`、`:large` 保存偏好；首次小屏默认卡片、大屏默认表格，跨断点自动切换。
- **执行对话框**：`param_map` 固定的参数（如 `cid`）不再出现在执行参数中，由所选条目自动注入；每个 action 声明自己的 params（挂时长不含 accuracy/include_completed，提交进度不含 duration）；参数区用 flex + `ga-4` 拉开间距，条目摘要改为 `primary-light` sheet；手动新建任务对话框同步整理间距。
- **验证**：大屏默认表格、小屏默认卡片；卡片标题为课程名且带进度环；执行对话框无课程 id 输入；提交进度只显示 unit_idx/accuracy/include_completed/limit，挂时长只显示 unit_idx/duration/limit；页面提交的任务 params 含 `cid=2416`，无未完成 SCO 时结果正确返回全 0。

## 37. 平台条目详情与脚本驱动展示（2026-11）
- **WeLearn 0.1.4**：发现新增 `scogeneral` 课程总览（练习总数、已完成、待完成、平均正确率、学习时长），并拆出 `get_course_identity` 避免发现时多拉单元；字段为 `total/done/remaining/avg_accuracy/study_time`，列 schema 同步更新。
- **展示由脚本决定**：manifest `discovery.columns[].important` 标记重要列；任一处标记时表格只显示重要列，其余列进入「详细信息」弹窗；未标记时沿用旧行为展示全部列。卡片视图展示全部字段和全部动作。前端不写死 WeLearn 字段名，`primary` 决定卡片标题。
- **交互**：表格行提供「动作」菜单和「详细信息」图标按钮；「动作」列出脚本全部动作，「详细信息」展示全部字段；卡片视图保留全部内容与动作。参数提示均来自脚本字段的 `help`。
- **后端**：`agenthost.Field` 与 `v1.PlatformField` 增加 `important`，发现/平台 schema 透传。
- **验证**：课程 566 返回 283 个练习、已完成 283、平均正确率 97%、学习时长 31 小时 7 分钟；表格只显示课程名/进度/状态/学习时长，详细信息弹窗显示课程ID/练习统计，卡片视图字段完整。

## 38. 表格操作、排序与挂时长总时长（2026-11）
- **表格操作**：移除「更多」展开行；表格行改为「动作」`v-menu`（列出脚本动作）和「详细信息」图标按钮（带 tooltip）。详细信息弹窗只展示字段，不再放动作执行按钮；卡片视图仍直接展示全部字段与动作。
- **表头排序**：使用 Vuetify `header.sortRaw`（条目字段在嵌套 `fields` 中，默认按 `f_*` 取不到值）。新增 `columns[].sort`（`number/percent/duration/date/text`）与通用比较器；学习时长按 `duration` 解析为秒，其他按数值/百分比/日期/自然文本兜底。实测学习时长升序 14 分钟→31 小时 7 分钟→50 小时 24 分钟→51 小时 35 分钟，降序相反。
- **后端**：`agenthost.Field` 与 `v1.PlatformField` 增加 `sort` 透传。
- **WeLearn 0.1.5**：`duration` 改为本次任务总挂时长，按本次 SCO 数均分（不再每个 SCO 都挂 duration 秒）；事件显示总时长与每个 SCO 的分配值。仓库提交 `3269213` 已推送。
- **真实测试**：课程 2416 `time`、`limit=2`、`duration=60`，两个 SCO 各 30 秒，任务 wall time 约 90 秒（含登录/单元请求），学习时长从 14 分钟涨到 15 分钟；`progress`（课程 566，limit=1，含已完成，正确率 100）成功提交 1 个 SCO。

## 39. 挂时长收敛与侧边栏置顶（2026-11）
- **WeLearn time 0.1.6**：总时长按约 60 秒一个 SCO 估算所需数量，只收集并处理这些 SCO；60 秒只处理 1 个，不再遍历全部 SCO 导致数分钟累积。真实测试课程 2417 学习时长 50 小时 24 分 → 50 小时 25 分，任务约 80 秒结束（含登录与课程请求）；进度动作课程 566 limit=1 含已完成成功提交。脚本提交 `9beedaf` 已推送。
- **侧边栏置顶**：任务二级菜单的平台右侧增加 pin 图标，可置顶/取消；置顶平台插入「概览」与「任务」之间，分组标题为 `Pin`。点击 Pin 内条目不会自动展开任务二级菜单。置顶项和任务子项使用手动 active 状态，同一平台同时出现时只高亮用户点击的那一个；pin 按钮不再带 tooltip/aria-label。
- **任务详情参数**：按脚本 manifest 的 action 参数标签展示；通过 `discovery.param_map` + 发现缓存把 `cid` 等参数解析为条目主字段（如课程名），并隐藏空值、0 限制和关闭的开关。课程参数用 primary 色 chip 展示。
- **存储**：新表 `user_preferences`（迁移 000011），`user_id` 主键 + `pinned_platforms` JSON；API `GET/PUT /api/v1/me/preferences`；Casbin 增加 user 权限；lite 模式 AutoMigrate 同步模型。
- **前端**：App.vue 加载用户偏好并乐观更新，失败回滚；置顶按钮带 aria-label/tooltip，点击阻止路由跳转；测试后已清空当前用户偏好。

## 40. 平台页标题与脚本反馈入口（2026-11）
- **标题**：`/platforms/:id` 的应用栏标题改为「平台名 + 脚本版本」（如 `WeLearn v0.1.6`），平台卡内副标题改为脚本作者。
- **反馈**：`PlatformResponse` 增加 `author`、`source_url`；应用栏在主题按钮左侧新增 bug 图标，仅当前平台来自 GitHub 时显示，跳转到脚本仓库的 `issues/new`。demo 等非 GitHub 来源不显示。
- **验证**：WeLearn 标题 `WeLearn v0.1.6`、作者 `CometNetworkStudio`、反馈链接 `https://github.com/CometNetworkStudio/welearn/issues/new`；首页无反馈按钮；demo 显示作者但不显示反馈。

## 41. 门户首页与官方免费实例（2026-11）
- **域名**：主域 `omnitasks.top` 部署静态门户；应用实例 `app.omnitasks.top`（Vue + Go 同源反代）。选 `app` 因短、通用，后续拆管理后台可用 `admin`。官方实例免费开放，同时保留自托管。
- **门户入口**：Header、Hero、部署区、CTA、Footer 均提供「在线使用」，统一指向 `https://app.omnitasks.top/`；部署区左侧为官方实例卡，右侧为 lite / full 自托管方式。
- **信任口径**：官方实例不主动查看、提取、导出或另作他用用户凭据；不出售、不共享数据；通过加密、权限隔离、日志脱敏与审计尽力防护，但不排除被黑客攻击等事件导致泄露，无法承诺绝不泄露；支持删除凭据与任务记录。页面明确「最稳妥仍是自己部署」，并建议设置独立密码。
- **使用规则**：官方实例仅限个人学习使用，禁止代刷牟利、批量注册、攻击或滥用；并发与频率做合理限制。
- **实现**：`C:\Users\zym\Desktop\omnitask-portal`（独立静态站，不并入主仓库）；纯 HTML + CSS，无运行期 JS、无外部字体或 CDN；沿用应用主题（primary `#1867C0`、背景 `#F4F6F8`、MD3 圆角与层次），支持系统深浅色。
- **品牌**：采用新七巧板 Logo；门户使用透明 `logo.png`，门户与前端 favicon 统一为白色圆角图标套件（ico / 32 / 180 / 192 / 512），原 chevron 内联标志已移除。
- **SEO**：canonical `https://omnitasks.top/`；sitemap/robots 同步真实域名；JSON-LD 增加 `Organization`、`WebSite`、官方实例 `SoftwareApplication` 与 8 条 FAQ。
- **验证**：本地静态服务 Lighthouse Accessibility / Best Practices / SEO 均为 100，Performance 99–100；390px 与 1440px 无横向溢出。



## 41. Cookie 会话鉴权与 P0 安全能力（2026-11）
- **鉴权**：完全移除 JWT，改为服务端会话。Cookie `omnitask_session` 为 `HttpOnly`、`SameSite=Lax`，`Secure` 按 TLS 或 `X-Forwarded-Proto` 自动判断；数据库只存令牌 SHA-256 摘要。禁用账号、修改角色、吊销会话都在下次请求立即生效。
- **CSRF**：非安全方法要求 `X-CSRF-Token` 与 `omnitask_csrf` 双提交 Cookie 一致，并校验 `Origin`；登录/注册只校验 `Origin`。
- **元数据**：新增公开 `GET /api/v1/meta`，返回注册模式、邀请码/验证码能力、会话与任务配额，前端按能力渲染注册页与限额提示。
- **注册与管理员**：`security.registration` 支持 `open/invite/closed`，`security.invite_code` 为邀请码；`security.allow_first_user_admin` 控制首个用户是否自动成为管理员；支持 `OMNITASK_ADMIN_USER`/`OMNITASK_ADMIN_PASSWORD` 启动时创建管理员。
- **登录防护**：连续失败达到 `security.login_max_failures`（默认 5）后在 `login_window_minutes`（默认 15）内拒绝登录；登录尝试写入 `login_attempts`，失败计数从最近一次成功登录之后重新计算，历史记录保留供审计。未知用户也执行一次 bcrypt，减少用户名枚举。
- **任务配额**：新增 `task.daily_tasks_per_user` 与 `task.max_pending_per_user`，0 表示不限；`tasks` 增加 `(user_id, created_at)` 索引。
- **脚本环境**：Host 不再把服务端完整环境变量传给脚本，只传系统白名单与 `OMNITASK_*`，避免主密钥、数据库配置泄漏。
- **日志**：`env=prod` 时关闭 GORM SQL 明细，避免令牌摘要、凭据密文与业务数据进入日志；同时生产环境默认关闭 Swagger。
- **管理安全页**：管理员侧边栏新增「安全」，可查看注册方式、会话数、失败登录、锁定与配额，吊销任意会话或某用户全部会话；普通用户可在设置页查看登录设备并退出其他设备。
- **迁移**：`000012_sessions` 创建 `sessions`、`login_attempts` 和任务索引；升级后旧 JWT 登录状态失效，需要重新登录。
- **验证**：Cookie 注册/登录/登出、CSRF 403/200、失败锁定 429、管理员吊销即时 401、安全页与会话列表均已在生产构建上验证。

## 42. 管理设置合并与可视化配置（2026-11）
- **设置页**：移除侧边栏「安全」入口，管理员的安全与配置能力合并到「设置」页新增的「站点安全」「站点配置」标签；普通用户保留「个人」「账号安全」「协议」。
- **站点安全**：展示注册方式、活跃会话、失败登录、锁定策略和任务配额，可吊销任意会话；卡片间距统一加大。
- **站点配置**：只暴露注册、会话、登录防护、任务配额、发现间隔等安全可热更新字段，不暴露主密钥、数据库、Redis、Agent 等敏感配置。
- **覆盖文件**：可视化修改写入主配置同目录的 `*.overrides.yml`（如 `deploy.overrides.yml`），不重写主配置、不丢失注释；启动时自动合并，已在 `.gitignore` 中排除。
- **热更新**：保存后立即应用到运行中的 viper，注册方式、邀请码、登录锁定、认证限流、会话 Cookie、任务配额、发现间隔均无需重启；会话有效期仅对新登录生效。监听地址、数据库、Agent、日志和密钥仍需重启，页面单独列出。
- **手动重载**：配置页提供「重新读取文件」，修改主配置或覆盖文件后可立即刷新可编辑字段，无需重启。
- **接口**：`GET /api/v1/admin/config`、`PUT /api/v1/admin/config`、`POST /api/v1/admin/config/reload`，仅管理员可用，更新日志只记录键名不记录值。
- **验证**：可视化修改登录失败窗口后 `/admin/security` 立即反映；覆盖文件只包含实际修改的键；主配置保持不变；设置页标签、站点安全数据与卡片间距均已在生产构建验证。

## 43. 协议、隐私与合规运营（2026-11）
- **协议版本**：用户协议与隐私政策统一为 `2026-11-26`，后端常量 `api/v1.CurrentTermsVersion` 通过 `/api/v1/meta` 暴露给前端。
- **注册同意**：注册表单增加必选勾选框；后端要求 `accept_terms=true` 且 `terms_version` 与当前版本一致，否则返回 400。用户表新增 `terms_version`、`terms_accepted_at`，迁移 `000013_user_terms`。
- **公开访问**：新增无需登录的 `/terms` 和 `/privacy`，登录页、注册页、设置页均有入口；已登录用户也可以直接访问，不会被重定向回首页。
- **设置页**：「协议」标签展示完整用户协议、隐私政策和投诉联系方式，替代原来的简短免责声明。
- **协议内容**：覆盖服务说明、账号安全、使用规则、第三方平台与脚本风险、凭据与数据、滥用投诉、违规封禁与申诉、服务终止、免责声明、数据收集与用途、保留期限、共享披露、用户权利、未成年人和联系方式。
- **门户**：`omnitask-portal/legal.html` 提供同一版本协议，门户导航和页脚增加协议、隐私、滥用投诉入口。
- **运营流程**：公开联系渠道为 QQ 频道 `https://pd.qq.com/s/N834h64oek`（频道号 `pd84113918`）；核验通常不超过 3 个工作日；可采取警告、限权、吊销会话、暂停或永久封禁、下架脚本、轮换邀请码和限制来源 IP；申诉窗口 7 日。
- **验证**：未登录访问 `/terms`、`/privacy` 正常；未勾选协议注册返回 400 且不建号；勾选后成功并写入版本与时间；门户 `legal.html` 可访问且包含协议、隐私和投诉内容。

## 44. 脚本 manifest 更名为 omnitask.json（2026-11）
- **决策**：项目刚开源，只有一个正式接入脚本，直接放弃 `script.json` 兼容，统一改为 `omnitask.json`，避免长期维护双文件名。
- **后端**：`agenthost.ManifestFileName` 统一常量；脚本扫描、版本校验、回滚、压缩包查找和错误提示全部使用 `omnitask.json`。
- **SDK**：`omnitask-sdk` 的 CLI 和 bridge 只读取 `omnitask.json`。
- **文档**：README、脚本协议、开发指南、接入指南和参数说明全部同步。
- **官方脚本**：WeLearn 仓库已将 manifest 重命名为 `omnitask.json` 并推送。
- **升级影响**：旧 Host 无法加载新脚本，旧脚本也无法被新 Host 加载；项目刚开源，当前只有 WeLearn 一个正式脚本，直接更新即可。新脚本目录必须包含 `omnitask.json`。
