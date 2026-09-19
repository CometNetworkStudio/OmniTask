# 参考项目分析（净室重写依据）

> 本文件只记录**行为与网络接口事实**，不包含、不移植任何原项目代码。
> 分析样本位于 `/tmp/opencode/refs/`，**不属于本仓库**。
>
> **协议来源说明（已获项目所有者授权）**：M1 阶段因缺少真实抓包样本，所有者同意
> 阅读参考仓库源码以**提取协议事实**（签名、jv 规则表、加密码算法、端点与参数顺序），
> 再**独立重写**实现。代码为本项目原创，未复制原项目代码；但严格“净室”主张因此减弱，
> 该权衡由所有者知晓并拍板。

## 授权核实结论

| 项目 | LICENSE | 结论 |
|------|---------|------|
| ularch/Easy_Cidaren | GPL-3.0（标准全文） | 传染性，禁止吸收代码 |
| jhl337/Auto_WeLearn | 无 LICENSE 文件 | 默认保留所有权利 |

→ 两项目**均只可参考行为，不可复制代码**；本项目全部独立实现，MIT 开源。

## 一、WeLearn（welearn.sflep.com）

原项目 `Auto_WeLearn`：PyQt5 GUI + `requests`，约 1.7k 行。UI 需全部剥离。

### 依赖（可保留的只有网络层）

- `requests`（保留，改为无界面）
- `PyQt5`（剥离，改由 Go 后端调度）

### 登录流程（需独立复现）

1. `GET https://welearn.sflep.com/user/prelogin.aspx?loginret=...` 跟随跳转，从最终 URL 提取 `code_challenge`、`state`。
2. 组装 `rturl=/connect/authorize/callback?...`。
3. 密码经自定义算法加密后，`POST https://sso.sflep.com/idsvr/account/login`，表单字段 `rturl/account/pwd/ts`。
4. 返回 JSON `code`：`1`=账号密码错误，`0`=成功。
5. 成功后 Cookie 落在 session，再次访问 `prelogin.aspx` 完成登录态。

### 密码加密算法（自 `core/crypto.py` 观察，须以报文独立验证后重写）

- 取毫秒时间戳 `T0`，`V=(T0>>16)&0xFF`，对密码每个字节 `V^=byte`，`remainder=V%100`，`T1=(T0//100)*100+remainder`。
- 明文串 `S = f"{T1}*" + hex(password)`，`base64(S)` 作为 `pwd`，`ts=T1`。

> ⚠️ 该算法直接来自原项目源码，严格净室应从真实报文推导。M3 实现前需确认是否接受"按此协议规范重写"。

### 业务接口

| 用途 | 方法 | 路径 | 关键参数 |
|------|------|------|----------|
| 获取课程 | GET | `/ajax/authCourse.aspx` | `action=gmc`，Referer `/2019/student/index.aspx` |
| 课程信息 | GET | `/student/course_info.aspx` | `cid`（正则取 `uid`/`classid`） |
| 课程单元 | GET | `/ajax/StudyStat.aspx` | `action=courseunits&cid&uid` |
| 单元 SCO | GET | `/ajax/StudyStat.aspx` | `action=scoLeaves&cid&uid&unitidx&classid` |
| 提交进度 | POST | `/Ajax/SCO.aspx` | `startsco160928` → `setscoinfo` → `savescoinfo160928` |
| 挂时长 | POST | `/Ajax/SCO.aspx` | `startsco160928` + 每 60s `keepsco_with_getticket_with_updatecmitime` + `savescoinfo160928` |

## 二、词达人（app.vocabgo.com）

原项目 `Easy_Cidaren`：PyQt6 + `mitmproxy` + `spacy`，约 ~17MB（含 NLP 模型）。UI、NLP 模型、翻译均需剥离。

### 关键架构事实

- **无账号密码登录**。原工具靠 `mitmproxy` 抓取微信小程序 Token（`api/token_capture_service.py`），或用户手动填入。
- 因此本项目**词达人平台必须以 Token 作为凭据输入**（契合 README「账号密码或 Token」）。
- Token 校验：`GET /student/api/Student/Main?timestamp=&version=2.6.1.231204&app_type=1`，返回 `code==1` 有效。

### 请求头要求

- `Host: app.vocabgo.com`
- `Abc: md5(User-Agent)`（自定义 UA，见原 `request_header.py`，需以真实抓包为准）
- `Authorization-V: cfcd208495d565ef66e7dff9f98764da`
- `Usertoken: <token>`
- `Accept-Encoding: br`（Brotli，需 `brotli` 解压）

### 已知接口

基础：`https://app.vocabgo.com/student/api/Student/`

| 用途 | 方法 | 路径 | 备注 |
|------|------|------|------|
| Token 校验 / 用户信息 | GET | `Student/Main` | `code==1` 有效 |
| 班级任务分页 | POST | `ClassTask/PageTask` | `task_type` 1 自学 / 2 班级测试 |
| 任务详情 | GET | `ClassTask/Info`、`StudyTask/Info` | 取 `task_id/release_id/task_name` |
| 取题 | GET | `{task_type}/StartAnswer` | 返回 `topic_code/stem/options/topic_mode` |
| 判分 | POST | `{task_type}/VerifyAnswer` | 返回 `answer_result`(1 对/2 错) 与 **`answer_corrects`（标准答案）** |
| 下一题 | POST | `{task_type}/SubmitAnswerAndSave` | 返回下一题加密数据 |
| 跳过 | POST | `{task_type}/SkipAnswer` | |
| 选词提交 | POST | `{task_type}/SubmitChoseWord` | 提示"需要选词"时用 |
| 查词 | GET | `Course/StudyWordInfo` | |
| 资源 CDN | — | `resource.vocabgo.com/Resource/CoursePage/` | |

### 协议机制

- **签名**：`md5(按序参数用 & 连接 + "ajfajfamsnfaflfasakljdlalkflak")`。
- **响应混淆**：`data` 为 base64，另有 `jv` 标识的**去噪混淆**（`JV_TWO` 删索引字符、`JV_THREE` 分块重排）；`check_jv` 会重试直到 `jv` 落在已知集合。看到 `jv` 不在已知集合时需保留样本。
- **风控**：`code=11003` 表示需在 App/微信端完成安全验证，重试无意义，应停止并提示人工。

### 答题策略（已修正）

1. **题库优先**：`question_key` 命中直接作答。
2. 未命中：**LLM** 作答（若配置了 DeepSeek）。
3. `VerifyAnswer.answer_corrects` 为权威答案，**回写题库**，下次直接命中。
4. 既无题库也未配置 LLM：`SkipAnswer` 跳过。

> **修正（重要）**：源码核对后确认 `StartAnswer.options[].answer_tag` 是每个选项的
> **提交码**（如 `0`/`1`，mode 41-44 甚至拼接为 `"0"+"2"="02"`），**不是“正确项”标记**。
> 原“用 answer_tag 猜正确答案”的设想不成立，已从策略中移除。

题型按 `topic_mode` 分支：`42`（选项，中文释义）、`32`（短语填空，按模板空位对齐）、`73`（空位单词）等。
净室实现独立设计，不复用原项目的 `answer_lib`、spacy 流程与谷歌翻译。

## 三、对 Agent 架构的映射

- 每个平台一个 `agent/scripts/<platform>/runner.py`，实现 `run(action, params, credentials)` 生成器（见 `agent/scripts/base.py`）。
- 凭据：
  - WeLearn：`{"username": ..., "password": ...}`
  - 词达人：`{"token": ...}`
- GUI 全部剥离；登录、抓取、答题逻辑改为无界面函数。
- 题库访问：Agent 不直连数据库，经后端 gRPC `QuestionBank` 服务读写（见 `docs/development-plan.md` §4/§5）。

## 四、已确认决策

1. WeLearn 密码加密算法：**按协议规范重写**（采纳，原作者已摸透平台规律）。
2. 词达人：**仅 Token 登录**（无密码登录，token 由用户抓包提供）。
3. 自动答题：**要做**，采用题库 → 接口答案 → LLM 的三级策略并回写题库。
