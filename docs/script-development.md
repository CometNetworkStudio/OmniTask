# OmniTask 脚本开发指南

> 面向平台脚本作者。目标：**最少改动接入已有项目，用平台提供的组件把执行过程展示清楚**。
>
> 协议细节：`docs/script-protocol.md` ｜ SDK：`sdk/python/README.md`

---

## 1. 五分钟接入

一个最小脚本只需要两个文件：

```
my-script/
├── script.json
└── main.py
```

`script.json`：

```json
{
  "protocol": 2,
  "id": "my-script",
  "name": "我的平台",
  "version": "0.1.0",
  "author": "you",
  "icon": "mdi-school",
  "runtime": "python3",
  "entry": "main.py",
  "actions": [
    { "id": "run", "name": "执行", "description": "完成平台上的待办任务" }
  ],
  "credentials": [
    { "key": "token", "label": "Token", "type": "password", "required": true }
  ],
  "params": [
    { "key": "course_id", "label": "课程", "type": "text", "placeholder": "留空表示全部" }
  ]
}
```

`main.py`：

```python
from omnitask_sdk import action, serve

@action("run")
def run(ctx):
    ctx.log("开始执行")
    ctx.progress(10, "登录成功")
    # 你的业务逻辑
    ctx.result["完成数量"] = 1

if __name__ == "__main__":
    serve()
```

发布到 GitHub 后，在 OmniTask「脚本」页填写仓库地址安装，Host 会热扫描生效。

---

## 2. 平台组件

所有组件都通过 `ctx` 调用，不需要写前端代码。

| 组件 | 调用 | 界面效果 |
|---|---|---|
| 进度条 | `ctx.progress(42, "正在处理")` | 顶部百分比进度 |
| 数量进度 | `ctx.progress_of(12, 144, "正在观看", unit="节")` | 进度 + `12/144 节` |
| 步骤清单 | `ctx.step("login", "登录", "succeeded")` | 步骤列表与状态图标 |
| 日志面板 | `ctx.log("token 即将过期", level="warn")` | 按级别着色的日志 |
| 指标卡片 | `ctx.metric("correct", "正确率", 96, "%")` | 最新指标标签 |
| 结果 | `ctx.result["score"] = 96` | 任务完成后的结果区 |

### 2.1 进度

```python
# 已知百分比
ctx.progress(35, "正在提交第 3 题")

# 已知数量和总量，百分比由平台计算
ctx.progress_of(done, total, "正在观看", unit="节", stage="watch")
```

约定：

- 百分比建议单调递增，不要回退。
- 长任务每 10 到 30 秒至少上报一次，避免界面看起来卡住。
- `current/total` 比单纯百分比更有信息量，优先使用。

### 2.2 步骤

```python
ctx.step("login", "登录", "running")
# ...
ctx.step("login", "登录", "succeeded")

ctx.step("submit", "提交作业", "failed")
```

状态取值：`running`、`succeeded`、`failed`、`skipped`。

同一个 `step` key 重复上报时，界面保留最新状态。建议 key 稳定、可读，例如 `login`、`fetch_courses`、`watch_video`。

### 2.3 日志

```python
ctx.log("开始处理课程 2416")
ctx.log("接口返回 429，稍后重试", level="warn")
ctx.error("登录失败")
```

级别：`debug`、`info`、`warn`、`error`。

不要把每道题、每个请求都写成日志，日志面板只保留最近 200 条；高频信息用进度和指标表达。

### 2.4 指标

```python
ctx.metric("correct", "正确率", 96, "%")
ctx.metric("remaining", "剩余任务", 12, "个")
ctx.metric("score", "当前得分", 88)
```

同一个 `key` 只展示最新值，适合展示正确率、剩余数量、得分等。

### 2.5 结果

```python
ctx.result["score"] = 96
ctx.result["detail"] = "完成 144 节"

# 也可以直接返回数组，配合 manifest 的 table 结果视图
ctx.result = [
    {"item": 1, "status": "完成:success"},
    {"item": 2, "status": "完成:success"},
]
```

在 `actions[].result` 声明渲染方式：

```json
"result": {
  "type": "table",
  "columns": [
    { "key": "item", "label": "项目", "type": "number" },
    { "key": "status", "label": "状态", "type": "chip" }
  ]
}
```

支持 `table`、`links`、默认键值（`keyvalue`）。`status` 里的 `完成:success` 会渲染为绿色标签。

### 2.6 取消

长任务可以在循环里检查取消：

```python
for i in range(total):
    ctx.check_canceled()
    ...
```

Host 取消时会先发送 `{"type":"cancel"}`，给脚本约 3 秒清理时间，再终止进程。`ctx.canceled` 也可以直接判断。

---

## 3. manifest v2 字段

### 3.1 顶层

| 字段 | 必填 | 说明 |
|---|---|---|
| `protocol` | 建议 | 当前为 `2` |
| `id` | 是 | 平台唯一标识，小写字母/数字/`-`/`_` |
| `name` | 是 | 展示名称 |
| `version` | 建议 | 语义化版本 |
| `icon` | 否 | MDI 图标名，如 `mdi-school` |
| `runtime` | 是 | 可执行文件，如 `python3`、`node` |
| `entry` | 二选一 | 入口文件 |
| `module` | 二选一 | `python -m module` 入口 |
| `actions` | 是 | 动作列表 |
| `params` | 否 | 平台级公共参数 |
| `credentials` | 否 | 凭据字段 |
| `permissions` | 否 | 权限声明（如 `network`、`filesystem`），仅透明展示 |
| `discovery` | 否 | 条目发现 schema |

### 3.2 action

```json
{
  "id": "study",
  "name": "刷课",
  "description": "完成课程视频与测验",
  "params": [
    { "key": "course_id", "label": "课程", "type": "select", "required": true,
      "options": [{ "label": "课程 A", "value": "2416" }] }
  ]
}
```

`actions[].params` 存在时优先使用动作级参数；否则回退到顶层 `params`。这样每个动作可以有独立表单。

`actions[].result` 声明结果渲染方式，见下一节。

### 3.3 参数与凭据字段

| 字段 | 说明 |
|---|---|
| `key` | 参数 key |
| `label` | 展示名 |
| `type` | `text`、`number`、`switch`、`select`、`textarea`、`password`、`token`、`cookie` |
| `default` | 默认值 |
| `required` | 是否必填，后端会校验 |
| `placeholder` | 输入提示 |
| `help` | 字段说明，显示在输入框下方 |
| `options` | `select` 的选项，支持字符串数组或 `{label,value}` |
| `min/max/step` | 数字输入约束 |
| `multiline` | 等价于 `textarea` |

账号级代理由平台在任务/发现时注入环境变量 `OMNITASK_PROXY`，SDK 的 `omnitask_sdk.http.new_session()` 会自动读取；如果脚本自己建 HTTP 会话，也请读取该变量。

凭据字段类型说明：

- `password`、`token`：前端遮罩输入；
- `cookie`：多行粘贴；
- 凭据只会通过 `ctx.credentials` 下发，不要落盘或打印。

### 3.4 discovery

```json
"discovery": {
  "label": "课程",
  "primary": "title",
  "columns": [
    { "key": "title", "label": "课程", "type": "text" },
    { "key": "progress", "label": "进度", "type": "progress" },
    { "key": "status", "label": "状态", "type": "chip" }
  ],
  "actions": ["study"],
  "param_map": { "course_id": "cid" }
}
```

`param_map` 会把选中条目的字段自动填入任务参数。列类型支持：

- `text`、`number`、`date`
- `progress`：`60%` 或 `0.6`
- `chip`：`已完成:success`
- `sparkline`：`1,2,3,5,4`

---

## 4. 非 Python 脚本

协议是逐行 JSON，任何语言都可以直接实现。Host 启动脚本后写入一行请求，脚本逐行输出事件。

请求：

```json
{"type":"execute","task_id":"...","platform":"my-script","action":"run",
 "params":{},"credentials":{}}
```

输出示例：

```json
{"type":"event","event":"started","message":"开始执行"}
{"type":"event","event":"step","step":"login","label":"登录","status":"running"}
{"type":"event","event":"progress","percent":20,"current":12,"total":60,"unit":"节","message":"正在处理"}
{"type":"event","event":"log","level":"warn","message":"触发限流，稍后重试"}
{"type":"event","event":"metric","key":"correct","label":"正确率","value":"96","unit":"%"}
{"type":"event","event":"succeeded","percent":100,"message":"done","result":{"score":"96"}}
```

约束：

- stdout 只能输出协议 JSON，一行一个；
- 调试信息走 stderr；
- 凭据只从请求读取；
- 结束前必须输出 `succeeded` 或 `failed`。

---

## 5. CLI 桥：零代码接入

已有命令行工具时，只写 `script.json`，无需写 Python：

```json
{
  "protocol": 2,
  "id": "my-cli",
  "name": "某 CLI",
  "runtime": "python3",
  "module": "omnitask_sdk.bridge",
  "actions": [
    { "id": "run", "name": "运行", "command": ["mycli", "run", "--course", "{course_id}"] }
  ],
  "credentials": [{ "key": "token", "label": "Token", "type": "password" }],
  "params": [{ "key": "course_id", "label": "课程", "type": "text", "required": true }]
}
```

命令输出可带组件标记：

```text
[progress] 42 正在处理
[step] login running 登录
[metric] correct 正确率 96 %
[log] warn 触发限流，稍后重试
```

其余输出按 `info` 日志展示。退出码 0 表示成功。

---

## 6. 平台内调试与更新

管理员在「脚本」页进入脚本详情，可以：

- 查看实时 manifest、动作、参数、结果视图、来源、ref 与 commit
- 点击「调试」直接填参数和凭据运行一次，查看组件事件；调试凭据只用于本次执行，不保存
- 点击「重新安装」按记录的来源与 ref 更新脚本
- 更新前会保留上一版本，可在详情页「回滚」
- 查看运行时状态与最近错误

脚本详情和目标平台页都以 Host 实际加载的 manifest 为准，修改 `script.json` 后无需重装即可刷新 schema。

## 7. 任务工作区与并发

平台默认**一个任务一个独立脚本进程**，并会把脚本源码复制到独立工作区：

```text
<workspaces_dir>/task-<task_id>/
  script.json
  main.py
  config.ini
  tmp/
  home/
```

因此：

- 你修改 `config.ini`、写 Cookie、写缓存，都只影响当前任务
- 不会和其他用户、其他账号的任务互相覆盖
- 运行时使用脚本原版本目录里的 `.venv`，不会复制依赖
- 大文件资源也会复制，脚本目录不要放超大文件

可用环境变量：

| 变量 | 含义 |
|---|---|
| `OMNITASK_WORKDIR` | 当前任务工作区 |
| `OMNITASK_SCRIPT_DIR` | 原脚本版本目录 |
| `OMNITASK_TASK_ID` | 任务 ID |
| `OMNITASK_PROXY` | 账号代理 |
| `OMNITASK_RATE_DB` | 跨进程请求间隔共享库，SDK `RateLimiter` 自动读取 |

脚本原有的相对路径写法通常不需要改；如果代码里用了绝对路径或写死了安装目录，请改为基于 `OMNITASK_WORKDIR` 或当前工作目录。

`.venv` 不会复制到工作区，运行时使用的是原版本目录里的解释器；脚本内部请用 `sys.executable`，不要用相对路径 `./.venv/bin/python` 启动子进程。

## 8. 本地开发

在脚本目录执行：

```bash
# 校验 manifest 与入口文件
python -m omnitask_sdk.cli check

# 模拟 Host 跑一次
python -m omnitask_sdk.cli run --action run --param course_id=2416 --credential token=xxx

# 输出 JSON Schema，供编辑器补全
python -m omnitask_sdk.cli schema
```

安装 SDK 后也可以直接使用：

```bash
omnitask-script check
omnitask-script run --action run
```

`run` 会打印组件事件、非 JSON 输出和 stderr，便于在安装到平台前发现问题。

---

## 9. 发现列表

如果需要让用户在平台页选择课程/作业，使用 `@listing`：

```python
from omnitask_sdk import listing, serve

@listing()
def courses(ctx):
    return [
        {"key": "2416", "fields": {"title": "高等数学", "progress": "60%", "status": "进行中:warning"}},
    ]

if __name__ == "__main__":
    serve()
```

配合 manifest 的 `discovery` schema，平台页会自动渲染表格、卡片、进度列和操作按钮。

如果发现结果需要筛选（例如学期、课程类型），在顶层 `params` 声明字段，平台页的「筛选」会通过 `DiscoverRequest.params` 传给 `@listing`：

```python
@listing()
def courses(ctx):
    semester = ctx.params.get("semester", "")
    return client.list_courses(semester)
```

---

## 10. 开发者注意事项

### 工作区

- 每个任务运行在独立工作区，当前工作目录就是工作区，不是脚本安装目录。
- 直接写 `./config.ini`、`./cookies.json`、`./cache.db` 只会影响当前任务。
- 不要写 `OMNITASK_SCRIPT_DIR` 或写死安装目录；它是原版本目录，仅作为只读参考。
- 需要明确路径时用 `OMNITASK_WORKDIR`：

```python
import os
from pathlib import Path

workdir = Path(os.environ.get("OMNITASK_WORKDIR", os.getcwd()))
config_path = workdir / "config.ini"
```

- `.venv` 不会复制到工作区。启动子进程请用 `sys.executable`，不要用 `./.venv/bin/python`。
- `HOME`、`USERPROFILE`、`TMPDIR`、`TEMP`、`XDG_CACHE_HOME`、`XDG_CONFIG_HOME` 都已指向工作区。临时文件用 `tempfile`，不要写死 `/tmp` 或用户主目录。
- 任务成功后工作区默认清理，失败任务默认保留。不要依赖工作区跨任务保存状态。
- 脚本目录会被复制到每个任务，尽量不要在里面放超大资源文件。
- 不要把凭据、代理密码等敏感信息写进工作区文件。

### 版本与更新

- 脚本安装使用版本化目录，`active.json` 由平台维护，不要手动修改。
- 同一脚本可以保留多个版本；运行中的任务使用自己的工作区副本，更新不会影响它。
- 不要在代码里依赖固定的安装路径或版本目录名。
- 回滚和更新由管理员在脚本详情页操作，脚本本身不需要处理。

### 并发与限流

- 平台默认一个任务一个独立脚本进程，不同任务不共享内存状态。
- 同一账号的任务会串行执行；不同账号、不同用户可以在平台/全局并发额度内并行。
- 脚本如果使用 SDK 的 `RateLimiter`，会自动读取 `OMNITASK_RATE_DB`，与同一 Host 上的其他进程共享请求间隔。
- 自己实现 HTTP 请求的脚本不会自动获得跨进程节流，建议使用 `omnitask_sdk.http.new_session()` 和 `RateLimiter`。
- 不要删除或直接修改 `OMNITASK_RATE_DB` 指向的 SQLite 文件。
- 长任务请在循环中调用 `ctx.check_canceled()`，不要忽略取消。
- 不要依赖“同时只有一个任务”这个假设；同一脚本可能被多个用户并发调用。

### 协议

- stdout 只能输出协议 JSON；调试信息走 stderr 或 `ctx.log()`。
- 结束前必须输出 `succeeded` 或 `failed`。
- 凭据只从 `ctx.credentials` 读取，不要打印或落盘。
- 代理由平台通过 `OMNITASK_PROXY` 注入；SDK 的 `new_session()` 会自动使用。
- 修改 `config.ini` 后直接运行即可，修改的是任务工作区副本。

## 11. 最佳实践

1. **一个 action 做一件事**。不要把“登录、刷课、交作业”塞进一个 action，步骤用 `ctx.step` 表达。
2. **进度用 current/total**，比百分比更能说明问题。
3. **日志不要刷屏**。高频细节用进度，异常用 warn/error。
4. **步骤 key 稳定**，方便界面合并状态。
5. **失败要给出可行动的信息**。例如“Token 已过期，请重新获取”，不要只抛出异常。
6. **不要打印凭据**，不要写临时文件保存凭据。
7. **脚本要能单独运行**，便于调试和排查。
8. **参数必填项要声明 `required`**，后端会在创建任务时拦截。
9. **生产安装固定 commit/tag**，避免上游分支变化导致行为漂移。
10. **长任务检查取消**，在循环中调用 `ctx.check_canceled()`。
11. **状态文件写在当前工作区**，不要写脚本安装目录或固定绝对路径；平台会为每个任务复制独立工作区。

---

## 12. 常见问题

**脚本显示“缺少运行时”**

服务器没有找到 manifest 里的 `runtime`。安装对应运行时，或在脚本目录自带 `.venv` / 二进制，并把 `runtime` 指向它。

**任务创建成功但执行失败**

先在脚本目录运行 `omnitask-script run`，确认 stdout 只有 JSON、最后有 `succeeded`/`failed`。

**界面没有步骤或日志**

检查是否调用了 `ctx.step` / `ctx.log`，并确认事件字段拼写正确。旧脚本需要按 v2 组件改造后才能获得完整展示。

**修改了 script.json 但界面没变**

重新安装脚本以刷新注册表 schema。
