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
| 仓库布局 | **Nunu advanced 作 `backend/`**，根目录另建 `frontend/`、`agent/`、`proto/` | 不采用 Nunu monorepo（其前端为 TS） |
| 前端脚手架 | **Vite `vue` 模板（纯 JS）** + Pinia + Vue Router + Axios | 忽略 Nunu 自带 web（TS） |
| 工具链版本 | Go 1.27.1 / Node 22.20.0 / Python 3.12 | Agent 用 `agent/.venv` |
| proto 生成 | `make proto`（Go→`backend/api/agent/v1`，Py→`agent/gen`），产物 gitignore | |
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
├── README.md              # 本文件（上下文存档）
├── Makefile               # make proto / agent-test / backend-run / frontend-dev
├── .gitignore
├── proto/                 # protobuf 定义（Go 与 Python 共享的唯一契约）
│   └── agent/v1/agent.proto
├── backend/               # Go 后端（Nunu advanced 模板，module = backend）
│   ├── api/agent/v1/      # protoc 生成，gitignore，make proto 重建
│   ├── cmd/{server,migration,task}/
│   ├── config/            # 默认 SQLite（local.yml），MySQL/Redis 配置预留
│   ├── internal/{handler,service,repository,model,middleware,server,job,task,router}/
│   ├── pkg/               # 含 pkg/server/grpc
│   ├── deploy/            # Dockerfile / docker-compose
│   └── storage/           # 运行期数据（gitignore）
├── frontend/              # Vue3 + JS + Vite + Pinia + Router + Axios
│   └── src/{router,api,views}/
├── agent/                 # Python Agent
│   ├── grpc_server.py     # gRPC 服务端（实现 AgentService）
│   ├── scripts/           # 各平台脚本（净室重写）
│   │   ├── base.py        # runner 接口约定
│   │   ├── cidaren/       # 词达人（M3）
│   │   └── welearn/       # WeLearn（M4）
│   ├── gen/agent/v1/      # protoc 生成，gitignore
│   ├── test_smoke.py      # gRPC 冒烟自检
│   └── requirements.txt
└── docs/
    ├── reference-analysis.md   # 参考项目行为/接口分析（净室依据）
    └── development-plan.md     # 开发规划：阶段/任务/验收/风险
```

> 说明：脚本与后端同处一个工作区（monorepo），通过 `proto/` 契约通信。

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
- [x] M2：后端领域层——User/Credential/Task/Question repository + service（bcrypt+JWT、AES-256-GCM 凭据、任务/题库）+ REST（`/v1/auth|credentials|tasks|questions`）+ GORM 迁移（MySQL 3380 / Redis 6350）；curl 验收通过，凭据确认为密文
- [x] M3：通讯层——Go gRPC 客户端（Agent 50051）+ `QuestionBank` gRPC 服务端（9000）+ 任务编排（pending→queued→running→终态、凭据解密、Redis 进度、TaskEvent）+ WebSocket `/v1/ws`；端到端联调通过（WebSocket 实时事件、成功/失败终态、题库 gRPC 读写）
- [x] M4：前端——登录/注册、凭据管理、任务列表+参数表单创建、任务详情（WebSocket 实时进度）、题库；Vuetify 4 + 主题 + Pinia + api/service 层，严格遵循前端 UI 规则（`npm run build` 通过，Vite 代理联调通过）
- [ ] M5：集成测试与加固
- [ ] M6：Docker 化部署与文档

### M0 验收命令

```bash
make proto          # 生成 Go/Python proto 代码（产物已 gitignore）
make agent-test     # Python Agent gRPC 冒烟测试
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
- 提交前运行：`make agent-test`、`cd backend && go build ./...`、`cd frontend && npm run build`；改 proto 后先 `make proto`。
- proto 生成产物（`backend/api/agent/`、`agent/gen/`）不入库，由 `make proto` 重建。
- Agent 依赖装在 `agent/.venv`，不污染系统 Python。

---

## 11. 免责声明

本项目仅供个人学习与技术研究使用。使用者需自行遵守各平台服务条款与学校相关规定，因使用本项目产生的一切后果由使用者自行承担。

---

## 12. 后续决策（M5+）

- **角色/权限**：仅 `admin` / `user` 两角色；后端引入 **Casbin**（RESTful model，`keyMatch2`），按 `(role, route, method)` 鉴权；**首个注册用户自动为 admin**；「脚本」页仅 admin 可见。前端同一套，按角色条件渲染。
- **数据库迁移**：改用 **golang-migrate**（`cmd/migrate`，`embed` SQL，`multiStatements`），AutoMigrate 仅保留本地快速开发。
- **README**：根目录改为面向 GitHub 的展示型 README；原上下文存档移至 `docs/context.md`；接入文档 `docs/integration-guide.md`。
- **脚本插件化方向**：`manifest(script.json)` + gRPC 契约（`AgentService.Execute` 已是通用）；仅**管理员**安装、固定 commit、安装前审核；脚本独立仓库、**不入主库并 gitignore**；恶意脚本风险由使用者自辨，平台声明免责。
- **前端规则**：MD3 圆角（`elevated`+`rounded=xl`，card `elevation=2`）、含输入对话框 `persistent`、统一 `v-snackbar-queue`（见 `AGENTS.md`）。
- **部署**：`./deploy.sh` 一键构建+迁移+重启；公网 `https://<your-host>:4096/omnitask/`；nginx 497 回退到 HTTPS。
