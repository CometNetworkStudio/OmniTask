# OmniTask 开发规划

> 本文是执行蓝图，配合`docs/context.md`（上下文存档）与 `docs/reference-analysis.md`（平台协议事实）阅读。
> 每完成一个阶段，回写本文件与 README 的 Roadmap。

---

## 1. 排序原则

1. **先攻坚不确定性最高的部分**：平台协议（词达人签名/jv 混淆、WeLearn SSO）是最大技术风险，先用 Python 独立跑通，避免后期推翻架构。
2. **契约先行**：proto 与数据库 schema 尽早冻结，前端/后端/Agent 可并行。
3. **每阶段可独立验收**：不依赖后续阶段即可证明"这一步对了"。
4. **保持三层边界**：前端只交互、后端只编排、Agent 只执行且**不直连数据库**。

用户建议顺序：`py → 后端 → 后端与 agent 通讯 → 前端 → 测试 → 上线`。
本规划**采纳**该顺序，仅做两点增强：
- 在 py 之前补一个 **P0.5 契约补全**（题库 RPC、schema），否则 py 阶段无法定义题库读写接口。
- 把"测试"贯穿各阶段（每阶段自带验收），P5 只做集成/E2E 与加固。

---

## 2. 现状（P0 与 P0.5 已完成）

- `backend/`：Nunu advanced，端口 8080，`go build/vet` 通过，示例代码已清理并重跑 wire。
- `frontend/`：Vite + Vue3 + JS + Pinia/Router/Axios，`npm run build` 通过。
- `sdk/python/`：平台脚本 Python SDK（`core/`、`runtime.py`、`gen/`、`tests/`），依赖在 `sdk/python/.venv`。
- `proto/agent/v1/agent.proto`：`AgentService.Health/Execute(server-streaming)` + `QuestionBank.Lookup/Save`。
- `backend/internal/model/model.go`：冻结 `User/Credential/Task/TaskEvent/Question` 五张表 schema。
- `.env.example`：约定主密钥、JWT、Agent/QuestionBank gRPC 地址、LLM、限流与脚本凭据。
- `sdk/python/core/`：公共层 config/events/http/ratelimit/bank/llm（单测通过）。
- 平台脚本：已迁出为独立仓库（`CometNetworkStudio/{cidaren,welearn}`），由 Go Host 以子进程运行；旧 `run_cli.py` 已移除。
- 文档：`README.md`、`docs/`（reference-analysis / development-plan / script-protocol / integration-guide / build-and-run 等）。

### M1 真机联调记录（凭据经 `.env` 注入，不入库）

- 词达人：`Student/Main` token 校验通过；班级学习任务全链路通（`PageTask → StudyTask/List → StudyTask/Info → SubmitChoseWord → StartAnswer → VerifyAnswer → SubmitAnswerAndSave`）。完成 **List 04（196/196，100%）**，题库回写 129 条；`mode 0` 为阅读卡需跳读。答题后加入人类思考延迟（`think_min/max` 默认 2s，`spend_min/max` 默认 5~15s）。
- WeLearn：账密登录成功；完成 **课程 2416 全部 144 个 SCO**。两种模式均验证：`time`（挂时长，2 个 SCO）与 `progress`（提交进度，其余）；写接口 `startsco`/`setscoinfo`/`savescoinfo` 均 `ret:0`。
- 修正 1：列表类接口 `data` 为明文 JSON，仅答题类接口为 jv/base64。
- 修正 2：WeLearn `unitidx` 必须是**单元数字下标（0-based）**，传单元 GUID 会导致 8 个单元返回同一批 18 个 SCO（144 条中仅 18 唯一）；修正后按 scoid 去重，144 个唯一 SCO。
- 术语：SCO = SCORM Shareable Content Object，WeLearn 课程的最小学习单元（视频/练习/听力等），刷时长与提交进度都按 SCO 操作。

---

## 3. 阶段总览

| 阶段 | 名称 | 产出 | 依赖 |
|------|------|------|------|
| P0.5 | 契约补全 | proto 题库 RPC、DB schema、配置约定 | P0 |
| P1 | Agent 平台协议层 | 词达人/WeLearn 无界面实现 + 本地 CLI 验证 | P0.5 |
| P2 | 后端领域层 | 用户/凭据/任务/题库 + REST + 迁移 | P0.5 |
| P3 | 通讯层 | Go gRPC 客户端 + 进度汇聚 Redis + WebSocket | P1、P2 |
| P4 | 前端 | 登录/凭据/任务/实时进度/题库管理 | P2、P3 |
| P5 | 集成与加固 | E2E、单测、限流、安全 | P1–P4 |
| P6 | 部署上线 | Docker Compose + nginx + 密钥管理 | P5 |

---

## 4. 逐阶段任务分解

### P0.5 契约补全（已完成）

**任务**
- proto 增加 `QuestionBank` service（Go 实现服务端，Agent 作为客户端调用）：
  - `rpc Lookup(LookupRequest) returns (LookupResponse)`：按 `platform + question_key` 查题库。
  - `rpc Save(SaveRequest) returns (SaveResponse)`：回写确认正确答案。
- 冻结数据库 schema（见 §5.1）。
- 约定配置来源：统一的 `.env` + 各服务 config（后端 viper yaml，Agent env）。

**验收**：`make proto` 生成双端代码；Go/Python 均可编译/导入；schema 评审通过。

**需确认 #1**：题库架构是否采用「后端 gRPC 服务 + Agent 客户端」，而非 Agent 直连 MySQL。

---

### P1 Agent 平台协议层（最高风险；代码已实现，待真实凭据联调）

**P1.0 公共层**
- `sdk/python/core/config.py`：读取 env/yaml（平台凭据、限流、LLM）。
- `sdk/python/core/http.py`：会话、UA、重试、超时、Brotli、代理。
- `sdk/python/core/events.py`：进度事件约定（已知 `scripts/base.py` 的 runner 接口）。
- `sdk/python/core/ratelimit.py`：平台级信号量 + 账号串行（全可配，默认平台并发 2）。
- `sdk/python/core/bank.py`：题库抽象接口；P1 用**本地 SQLite/JSON 适配器**独立验证，P3 替换为 gRPC 适配器。
- `sdk/python/core/llm.py`：OpenAI 兼容客户端（`base_url/api_key/model` 可配），仅词达人使用。

**P1.1 词达人（`scripts/cidaren/`）**
- 协议层：`sign = md5(按序参数 & 连接 + "ajfajfamsnfaflfasakljdlalkflak")`；响应 `jv` 混淆 + base64 解码（`JV_TWO/JV_THREE`）。
- 接口：`Student/Main`（校验 token）、`ClassTask/PageTask`、`{type}/StartAnswer`、`{type}/VerifyAnswer`、`{type}/SubmitAnswerAndSave`、`{type}/SkipAnswer`、`{type}/SubmitChoseWord`、`Course/StudyWordInfo`。
- **答题策略（三级）**：
  1. 题库命中（`bank.Lookup`）直接作答；
  2. 未命中：优先使用 `StartAnswer.options[].answer_tag`（**需实测确认其是否标记正确项**），其次 LLM；
  3. `VerifyAnswer` 返回 `answer_corrects` 后**回写题库**（权威答案）。
- 题型分支：`topic_mode` 42（选项）、32（短语填空）、73（空位单词）等，按 options/模板对齐。
- 风控：识别 `code=11003`（需安全验证）→ 停止并上报人工介入；请求间隔随机化。

**P1.2 WeLearn（`scripts/welearn/`）**
- 登录：`prelogin.aspx` 取 `code_challenge/state` → 密码加密 → `sso.sflep.com/idsvr/account/login` → 落 Cookie。
- 业务：`authCourse`、`course_info`、`StudyStat(courseunits/scoLeaves)`、`SCO.aspx(startsco/setscoinfo/savescoinfo/keepsco)`。
- 两种动作：刷时长（按配置时长 + 每 60s keepalive）、提交进度。

**P1.3 本地验证**
- （历史）曾用 `agent/run_cli.py` 本地验证；该入口已移除，改用脚本自带 CLI 与 Go Host。
- **当前无可用 token**：先用脱敏样本写单元测试（签名、jv 解码、答题分支）；拿到 token 后再跑真实取题/判分/回写。

**验收**：命令行独立完成词达人与 WeLearn 各一次任务；题库有新增记录；不依赖 Go 后端。（真实 token 到位前，以单测 + 脱敏回放通过为阶段验收）

**需确认 #2/#3**：LLM 供应商与 key；提供可用词达人 token 做 `answer_tag` 实测。

---

### P2 后端领域层（已完成）

**已实现**
- `internal/repository/domain.go`：`UserRepository` / `CredentialRepository` / `TaskRepository` / `QuestionRepository`（GORM，凭据/题库 upsert）。
- `internal/service/{auth,credential,task,question}.go`：bcrypt+JWT 注册登录；AES-256-GCM 凭据加解密（`pkg/crypto`，密钥 `OMNITASK_MASTER_KEY`）；任务 CRUD；题库 Lookup/Save/List。
- `internal/handler/domain.go` + `router` + `server/http.go`：`/api/v1/auth/register|login`、`/api/v1/credentials`、`/api/v1/tasks`、`/api/v1/questions(+/lookup)`；JWT 中间件保护。
- 迁移接入 5 张表；配置切到 MySQL(3380) `omnitask` + Redis(6350)；wire 重新生成。

**验收记录**
- `docker compose up -d` 起 MySQL/Redis；`omnitask-server -migrate` 建表成功（users/credentials/tasks/task_events/questions）。
- curl 全链路：注册 → 登录 → 凭据入库 → 建任务 → 查任务 → 题库存/查；未带 token 返回 401。
- 凭据密文验证：`credentials.cipher` 48B、`nonce` 12B，库中无明文。
- `go build/vet/test` 通过；新增 `pkg/crypto` 单测（往返、错误密钥、坏 nonce）。
- 修复：MySQL JSON 列不接受空串，Task.Result 与 Question.Options/Answer 兜底为 `{}`/`[]`。

**模型（`internal/model/`）**：`User`、`Credential`、`Task`、`TaskEvent`、`Question`。
**repository**：GORM + gorm/gen；`CredentialRepository`、`TaskRepository`、`QuestionRepository`、`UserRepository`。
**service**
- `auth`：注册/登录（bcrypt）+ JWT。
- `credential`：AES-256-GCM 加解密，密钥 `OMNITASK_MASTER_KEY`（env）；入库密文+nonce。
- `task`：增删查、状态机。
- `question`：Lookup/Save、去重（`platform+question_key` 唯一）。
**handler/router**：`/api/v1/auth/*`、`/api/v1/credentials`、`/api/v1/tasks`、`/api/v1/questions`。
**迁移**：`internal/migration`（`server -migrate`）；配置切到 MySQL(3380)+Redis(6350)，用 `backend/deploy/docker-compose`。

**验收**：curl 完成注册/登录/建任务/查任务/题库增查；数据库确认凭据为密文；`make` 迁移成功。

---

### P3 通讯层（已完成）

**已实现**
- `internal/service/agent.go`：Go gRPC 客户端（`grpc.NewClient`，Health / Execute server-streaming）。
- `internal/server/bank.go`：`QuestionBank` gRPC 服务端（Go，端口 9000），Lookup/Save 直连题库。
- `internal/service/orchestrator.go`：轮询 pending → `Claim` 原子领取 → running → 解密凭据 → 下发 Agent → 终态；事件写 Redis（`ProgressStore`）、`task_events`、WebSocket。
- `internal/ws/hub.go` + `handler/ws.go`：`/api/v1/ws?task_id=` 订阅，连上先回放 Redis 快照。
- Agent：`core/bank.py` 新增 `GrpcBank`，`OMNITASK_BANK_MODE=grpc` 时经后端读写（Agent 不直连 DB）。
- wire 接入 Redis/AgentClient/ProgressStore/Hub/Orchestrator/BankGRPCServer。

**验收记录**
- 端到端（WeLearn，无副作用）：REST 建任务 → orchestrator 领取 → 从 DB 解密账密 → gRPC 下发 Agent → WebSocket 依次收到 `running(start/登录成功/课程数/没有未完成的 SCO)` → `succeeded`。
- 失败路径（词达人）：Agent 侧返回 `SecurityVerifyError`（11003 需安全验证）→ `failed` 事件经 WS 推送并落 `task_events`。
- `QuestionBank` gRPC 直连验证：Lookup 命中/未命中、Save 写入、再次 Lookup 命中并 `hit_count+1`；MySQL `questions` 落库。
- **待办**：词达人 token 触发平台风控（11003），需在 App/微信完成安全验证后重试；届时可验证完整答题+题库回写闭环。

- proto：`AgentService` 已有；P0.5 的 `QuestionBank` 在此实现 Go 服务端，并让 Agent 以客户端接入。
- Go gRPC 客户端：`internal/service/agent`（dial Agent@50051，`Execute` 流式）。
- 任务编排：消费者取 `pending` 任务 → 组装 `ExecuteRequest` → 流式接收事件 → 写 Redis 进度 + `TaskEvent` 落库 → 终态更新。
- WebSocket：`/ws` 订阅 `task_id`，事件驱动推送（后端→前端）。
- 并发/限流：后端侧任务队列 + Agent 侧平台信号量。

**验收**：从后端 REST 发起任务，Agent 执行，`websocat`/前端实时收到进度，终态与题库正确落库。

---

### P4 前端（已完成）

**已实现**（严格遵守 `AGENTS.md` 前端 UI 规则）
- 基础设施：Vuetify 4 + 自定义 Theme（`main.js`，颜色/圆角/默认值走 Theme）、Pinia、Vue Router（登录守卫）、Axios 封装 `api/http.js`（JWT 注入、响应解包、401 跳登录）、Vite 代理 `/api`（含 ws）。
- 分层：`api/services/{auth,credentials,tasks,questions}.js`；页面不直接使用 axios。
- Store：`stores/auth.js`（token/用户，localStorage 持久化）、`stores/ui.js`（snackbar）。
- 页面：登录、注册、任务列表（`v-data-table` + 新建任务对话框，参数按平台 schema 动态渲染，覆盖 `docs/parameters.md`）、任务详情（`/api/v1/ws` 实时进度时间线）、凭据管理（按平台动态字段）、题库（平台筛选 + 表格）。
- `src/config/platforms.js`：平台/动作/参数/凭据字段 schema；`src/ws/taskSocket.js`：WebSocket 客户端。
- 合规检查：无内联 `style=`、无 `<style>`、页面无 axios、组件无硬编码颜色。

**验收记录**
- `npm run build` 通过。
- `npm run dev` + 后端联调：`/api/v1/tasks` 经 Vite 代理返回 401 `{code:401}`，代理与鉴权链路正常。

- 页面：登录/注册、凭据管理（平台 + token/账号密码）、平台/任务类型选择、任务创建、任务详情（实时进度）、题库查看。
- 结构：`src/api/*`、`src/stores/*`（Pinia）、`src/router/*`、`src/ws/*`。
- 组件库：**Vuetify 4**（已安装，官方 MCP 已接入）。

**验收**：浏览器完成"登录 → 录入凭据 → 建任务 → 看进度 → 看结果"全流程。

---

### P5 集成与加固

- 单测：Python（签名/解码/答题策略，纯函数优先）、Go（service）。
- E2E：`docker compose` 起全套，脚本化跑一次任务。
- 安全：密钥不入库/不入日志；接口鉴权；输入校验；凭据解密仅在下发瞬间。
- 稳定性：重试、幂等、任务超时、Agent 掉线处理。

**验收**：CI 命令一键跑通；无明文凭据泄漏路径。

---

### P6 部署上线

- `deploy/docker-compose/Dockerfile`：多阶段构建前端 + proto + server + agent。
- `deploy/docker-compose/docker-compose.full.yml`：mysql/redis/server/agent/nginx。
- 配置模板首启复制到 Docker 卷；密钥由 server 首次生成；数据卷持久化。
- nginx：内嵌前端反代 + `/api` + WebSocket 升级。

**验收**：目标服务器 `docker compose up -d` 后全流程可用。

---

## 5. 关键设计

### 5.1 数据库表（已冻结，见 `backend/internal/model/model.go`）

- `users(id, username unique, password_hash, role, created_at, updated_at)`
- `credentials(id, user_id, platform, cipher, nonce, created_at, updated_at, unique(user_id, platform))`
- `tasks(id, user_id, platform, action, params json, status, progress, result json, error, started_at, finished_at, timestamps)`
- `task_events(id, task_id, type, percent, message, created_at, index(task_id))`
- `questions(id, platform, question_key, stem, options json, answer json, source enum(bank,api,llm), confidence, hit_count, created_at, updated_at, unique(platform, question_key))`

`question_key` = 对「标准化题干 + 题型」做哈希；测试阶段先用 `stem.content + topic_mode`，实现后按实测调整。

### 5.2 凭据加密

AES-256-GCM，随机 nonce 随密文存储；解密仅在下发 Agent 指令时进行；任何日志禁止出现明文。

- **密钥来源**：`security.master_key`，其次 `OMNITASK_MASTER_KEY`；默认版本 1。
- **版本与轮换**：`security.master_keys` 为 `版本号 -> 密钥` 映射，`credentials.key_version` 记录密文所用版本；映射非空时只读该映射，不再读取 `master_key`。新写入使用最大版本（可用 `security.master_key_version` 显式指定）。轮换步骤：把旧 `master_key` 以原版本号加入 `master_keys`，再加入新版本；执行 `server -rotate-credentials` 将全部凭据重加密到当前版本；确认后从映射移除旧版本。
- **审计**：每次任务/发现/编辑触发的解密写 `credential_access_logs`，并更新 `credentials.last_used_at`；`security.credential_audit_fail_closed` 为 true 时，审计写失败则拒绝解密使用。
- **日志**：请求/响应中间件不记录正文，查询串中的 token/password 等字段脱敏。

### 5.3 任务状态机

`pending → queued → running → succeeded | failed | canceled`；每次状态变更写 `task_events` 并推 WebSocket。

### 5.4 三级答题

优先级：**题库(bank)** → **接口提示/answer_corrects(api)** → **LLM**。
只有 `VerifyAnswer` 返回的 `answer_corrects` 标记为权威来源（`source=api`，高置信）；LLM 答案标记为待验证，命中成功后再升级。题库优先可显著省 token。

### 5.5 端口分配（本机现状）

| 服务 | 端口 |
|------|------|
| 后端 HTTP | 8080 |
| Agent gRPC | 50051 |
| MySQL | 3380 |
| Redis | 6350 |
| 前端 dev | 5173 |
| nginx（prod） | 80/443 |

> 本机 8000/3000 已被其他容器占用，故不用 Nunu 默认 8000。

---

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| 平台风控（词达人 11003 安全验证） | 任务中断 | 识别即停止并提示人工；低频、随机间隔；限流可配 |
| 词达人 token 时效/抓包成本 | 需频繁更新凭据 | 支持手动更新 token；校验失败提前报错 |
| jv 混淆出现新版本 | 解码失败 | 解码失败重试/上报；保留未知 jv 样本 |
| LLM 费用/准确率 | 成本或正确率 | 题库优先、命中缓存、prompt 精简、可关 |
| 账号封禁 | 服务不可用 | 默认低并发、单账号串行、仅个人少账号 |
| 净室合规 | 授权风险 | 不复制代码，保留独立实现证据（见 reference-analysis） |
| `answer_tag` 不可靠 | 答题策略降级 | 实测确认；不可靠则题库+LLM 为主 |

---

## 7. Definition of Done（通用）

- 代码通过 lint/构建：`go build ./...`、`go vet ./...`、`npm run build`、Python 冒烟/单测。
- 不引入明文密钥；不新增未使用依赖。
- 重要改动回写 README Decision Log / Roadmap 与本文件。
- 每个阶段的验收命令可一键复现。

---

## 8. 已确认决策（原开放问题）

1. **题库架构**：由 Go 后端暴露 gRPC `QuestionBank`（`Lookup`/`Save`），Agent 作为客户端调用；Agent 独立测试时用本地 SQLite/JSON 适配器。✅
2. **LLM 供应商**：**DeepSeek**（OpenAI 兼容）；`base_url/api_key/model` 由环境变量注入，key 由用户提供。✅
3. **词达人 token**：暂时无法提供 → P1 先写实现 + 单元测试（用脱敏样本），真实 token 的 `answer_tag` 实测与联调后置。⏳
4. **前端组件库**：**Vuetify**（最新 4.2.1，`npm i vuetify`），官方 MCP `@vuetify/mcp` 已接入 opencode 全局配置。✅
5. **WeLearn 风控**：账号无图形验证码，直接账密登录。✅

> 注意：Vuetify 最新为 **4.2.1**（2026-09-09 发布，较新），而官方文档/MCP 目前主推 3.13 LTS。若 v4 遇到 API 不稳定，可回退 `npm i vuetify@3`（本项目尚未写业务 UI，回退成本低）。

---

## 9. 脚本插件化（方案 C：Go Host + stdio JSON，第一阶段已完成）

**已完成**
- 协议文档 `docs/script-protocol.md`（manifest + NDJSON 事件 + 运行时检测 + gRPC 映射）。
- Go Host：`backend/internal/agenthost`（扫描脚本目录、manifest 解析、绝对路径解析、子进程 + stdio JSON 转发、事件映射），入口 `cmd/agent`，实现既有 `AgentService`，后端零改动。
- Python 便捷运行时 `sdk/python/runtime.py`（`serve(run_func)`）；`agent/scripts/<id>/{main.py,script.json}`。
- 配置：`agent.{host,port,scripts_dir,python,pythonpath,bank_addr}`；`omnitask.sh deploy` 改为构建并运行 Go Host（不再跑 Python gRPC server）。
- 验证：`Health` 返回 `['cidaren','welearn']`；`Execute(welearn, progress)` 子进程执行成功（登录→课程→完成→SUCCEEDED）。

**待办（下一阶段）**
- 脚本注册表（DB 表）+ 安装 API（GitHub zip → `scripts/<id>/`，固定 commit、加速源由前端拼接）+ 前端「脚本」页（列表/安装/启停/卸载）。
- `SupportedPlatforms` 改为读取注册表/ Host Health，而非硬编码。
- 运行时缺失的**前端状态展示**（Host 预检结果上报）。
- 内置 `cidaren/welearn` 拆到独立仓库并 `gitignore` 运行目录脚本。

### 9.1 脚本安装/注册表（已完成）
- `scripts` 注册表（迁移 000002）、`ScriptService`（下载 zip + 解压 + 校验 runtime + 注册）、REST（`/api/v1/scripts` 系列，仅 admin）、前端「脚本」页。
- 运行目录 `scripts/`（gitignore）；Host 热扫描；`README`/`integration-guide` 指向独立脚本仓库。
- 验证：从 GitHub 安装 cidaren/welearn → Host Health 识别 → welearn 任务经安装脚本执行成功。

### 9.2 lite/full 双模式（已完成）
- `config.yaml` 统一配置（自动生成、`mode` 默认 lite、密钥明文）；lite=SQLite+无 Redis+AutoMigrate；full=MySQL+Redis+golang-migrate。
- `pkg/config` 处理 mode/密钥/目录；`crypto.FromConfig`；`repository.NewRedis` 在 lite 返回 nil；`ProgressStore` 支持内存。
- 验证：临时目录运行 `-conf config.yaml` → 自动生成配置与 data/logs/scripts → 注册/登录成功 → SQLite 含 `users/credentials/tasks/task_events/questions/scripts/casbin_rule`。
- ✅ `go:embed` 前端 dist 单二进制静态托管；API 前缀改为 `/api/v1`。
