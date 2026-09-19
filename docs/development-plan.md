# OmniTask 开发规划

> 本文是执行蓝图，配合根目录 `README.md`（上下文存档）与 `docs/reference-analysis.md`（平台协议事实）阅读。
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
- `agent/`：gRPC 服务端骨架 + `scripts/{cidaren,welearn}` + 冒烟测试通过；依赖在 `agent/.venv`。
- `proto/agent/v1/agent.proto`：`AgentService.Health/Execute(server-streaming)` + `QuestionBank.Lookup/Save`。
- `backend/internal/model/model.go`：冻结 `User/Credential/Task/TaskEvent/Question` 五张表 schema。
- `.env.example`：约定主密钥、JWT、Agent/QuestionBank gRPC 地址、LLM、限流与本地 CLI 凭据。
- `agent/core/`：公共层 config/events/http/ratelimit/bank/llm（P1.0，单测通过）。
- `agent/scripts/cidaren/`：sign/jv/client/answer/runner（P1.1）。
- `agent/scripts/welearn/`：crypto/client/runner（P1.2）。
- `agent/run_cli.py`：本地命令行验证入口（P1.3）。
- 文档：`README.md`、`docs/reference-analysis.md`、`docs/development-plan.md`。

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
- `agent/core/config.py`：读取 env/yaml（平台凭据、限流、LLM）。
- `agent/core/http.py`：会话、UA、重试、超时、Brotli、代理。
- `agent/core/events.py`：进度事件约定（已知 `scripts/base.py` 的 runner 接口）。
- `agent/core/ratelimit.py`：平台级信号量 + 账号串行（全可配，默认平台并发 2）。
- `agent/core/bank.py`：题库抽象接口；P1 用**本地 SQLite/JSON 适配器**独立验证，P3 替换为 gRPC 适配器。
- `agent/core/llm.py`：OpenAI 兼容客户端（`base_url/api_key/model` 可配），仅词达人使用。

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
- 新增 `agent/run_cli.py`：`python run_cli.py --platform cidaren --action study`，从 env 读 token/账号。
- **当前无可用 token**：先用脱敏样本写单元测试（签名、jv 解码、答题分支）；拿到 token 后再跑真实取题/判分/回写。

**验收**：命令行独立完成词达人与 WeLearn 各一次任务；题库有新增记录；不依赖 Go 后端。（真实 token 到位前，以单测 + 脱敏回放通过为阶段验收）

**需确认 #2/#3**：LLM 供应商与 key；提供可用词达人 token 做 `answer_tag` 实测。

---

### P2 后端领域层

**模型（`internal/model/`）**：`User`、`Credential`、`Task`、`TaskEvent`、`Question`。
**repository**：GORM + gorm/gen；`CredentialRepository`、`TaskRepository`、`QuestionRepository`、`UserRepository`。
**service**
- `auth`：注册/登录（bcrypt）+ JWT。
- `credential`：AES-256-GCM 加解密，密钥 `OMNITASK_MASTER_KEY`（env）；入库密文+nonce。
- `task`：增删查、状态机。
- `question`：Lookup/Save、去重（`platform+question_key` 唯一）。
**handler/router**：`/v1/auth/*`、`/v1/credentials`、`/v1/tasks`、`/v1/questions`。
**迁移**：`cmd/migration` 接入全部模型；配置切到 MySQL(3380)+Redis(6350)，用 `backend/deploy/docker-compose`。

**验收**：curl 完成注册/登录/建任务/查任务/题库增查；数据库确认凭据为密文；`make` 迁移成功。

---

### P3 通讯层

- proto：`AgentService` 已有；P0.5 的 `QuestionBank` 在此实现 Go 服务端，并让 Agent 以客户端接入。
- Go gRPC 客户端：`internal/service/agent`（dial Agent@50051，`Execute` 流式）。
- 任务编排：消费者取 `pending` 任务 → 组装 `ExecuteRequest` → 流式接收事件 → 写 Redis 进度 + `TaskEvent` 落库 → 终态更新。
- WebSocket：`/ws` 订阅 `task_id`，事件驱动推送（后端→前端）。
- 并发/限流：后端侧任务队列 + Agent 侧平台信号量。

**验收**：从后端 REST 发起任务，Agent 执行，`websocat`/前端实时收到进度，终态与题库正确落库。

---

### P4 前端

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

- `Dockerfile`：backend、agent；compiler 产物多阶段构建。
- `docker-compose.yml`：mysql/redis/backend/agent/frontend+nginx。
- 环境变量与密钥注入；数据卷持久化。
- nginx：静态前端 + `/api` 反代 + `/ws` 升级。

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

AES-256-GCM，`OMNITASK_MASTER_KEY`（32 字节，hex/base64）来源 env；随机 nonce 随密文存储；解密仅在下发 Agent 指令时进行；任何日志禁止出现明文。

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
