# OmniTask 脚本协议（Host ↔ 脚本，stdio JSON）

> 目标：脚本是**独立项目**（任意语言），既能本地单跑（CLI/GUI），也能被 OmniTask Host 以子进程方式调用。
> Host 用 Go 编写，统一负责生命周期与调度，并沿用现有 gRPC `AgentService` 与后端通讯。

## 1. 总体架构

```
浏览器 ── REST/WS ── Go 后端(server) ──gRPC AgentService──▶ Go Host(agent)
                                                              │ 子进程 + stdio JSON
                                                              ▼
                                                     脚本(任意语言) ──gRPC QuestionBank──▶ 后端
```

- 后端只连 Host（`agent.addr`），不直接管脚本。
- Host 扫描脚本目录，按 manifest 启动/调用脚本子进程。
- 脚本可通过后端 gRPC `QuestionBank` 读写题库（若需要）。

## 2. 脚本目录与 manifest

版本化布局（推荐，Host 以 `active.json` 指向的版本运行）：

```
<scripts_dir>/
└── <script-id>/
    ├── active.json              # {"active":"<version>","previous":"<version>"}
    └── versions/
        └── <version>/
            ├── omnitask.json      # 必须
            ├── main.py          # entry（示例）
            └── ...
```

兼容旧的扁平布局 `<scripts_dir>/<script-id>/omnitask.json`；安装新脚本时自动使用版本化布局。

`omnitask.json`（推荐完整写法）：

```json
{
  "protocol": 2,
  "id": "welearn",
  "name": "WeLearn",
  "version": "0.2.0",
  "author": "CometNetworkStudio",
  "icon": "mdi-school",
  "runtime": "python3",
  "entry": "main.py",
  "module": "",
  "args": [],
  "requires": ["requests"],
  "progress_regex": "",
  "actions": [
    {
      "id": "time",
      "name": "挂时长",
      "description": "完成课程视频时长",
      "params": [
        { "key": "duration", "label": "挂时长(秒)", "type": "number", "default": 60 }
      ],
      "result": {
        "type": "table",
        "columns": [
          { "key": "course", "label": "课程", "type": "text" },
          { "key": "status", "label": "状态", "type": "chip" }
        ]
      }
    },
    {
      "id": "progress",
      "name": "提交进度",
      "description": "按课程提交学习进度",
      "command": ["mycli", "progress", "--cid", "{cid}"]
    }
  ],
  "credentials": [
    { "key": "username", "label": "账号", "type": "text", "required": true },
    { "key": "password", "label": "密码", "type": "password", "required": true }
  ],
  "params": [
    { "key": "cid", "label": "课程 id", "type": "text" }
  ]
}
```

- `runtime`：可执行名（`python3`、`node`、或脚本自带二进制路径）。
- `entry` / `module`：**二选一，均不限定文件名**。
  - `entry`：相对脚本目录的入口文件，Host 以 `runtime <entry> args...` 启动；
  - `module`：Host 以 `runtime -m <module> args...` 启动（Python 模块）。
  - **你不需要把文件叫 `runner.py`/`main.py`**——那只是官方脚本的示例。
- `args`：附加命令行参数。
- `requires`：运行时依赖说明，供展示与预检；实际安装读取脚本目录的 `requirements.txt`（需开启 `agent.auto_install_deps`）。
- `permissions`：权限声明（如 `network`、`filesystem`），目前仅透明展示，不强制执行。
- `progress_regex`：CLI 桥提取进度的正则（捕获组 1=百分比，2=消息），可空。
- `actions[].command`：**仅 CLI 桥**使用；该动作要执行的命令（支持 `{占位符}`）。
- `actions[].description`：动作说明，显示在执行对话框。
- `actions[].params`：动作级参数；存在时优先于顶层 `params`，未声明则回退。
- `actions[].result`：结果渲染 schema，`type` 支持 `keyvalue`（默认）、`table`、`links`；`columns` 用于 table 表头。
- 字段支持 `required`、`placeholder`、`help`、`options`（select）、`min/max/step`（number）、`multiline`。
- 用 `@action` 装饰器或生成器接入时无需 `module`/`command`，见 `sdk/python/README.md`。
- 安装时会校验 manifest：id、runtime、entry/module、action/字段 key 唯一性；v2 至少一个 action。

## 3. 运行协议（NDJSON）

Host 启动脚本后，**一个任务一个子进程**：

1. Host 向脚本 **stdin** 写一行请求 JSON：
   ```json
   {"type":"execute","task_id":"L8...","platform":"welearn","action":"time",
    "params":{"cid":"2416","duration":"60"},
    "credentials":{"username":"...","password":"..."},
    "bank_addr":"127.0.0.1:9000"}
   ```
2. 脚本向 **stdout** 逐行输出事件 JSON。v2 事件类型：

   | event | 说明 | 关键字段 |
   |---|---|---|
   | `started` | 开始执行 | `message` |
   | `progress` | 百分比或数量进度 | `percent`、`current`、`total`、`unit`、`stage` |
   | `step` | 步骤状态 | `step`、`label`、`status` |
   | `log` | 日志 | `level`、`message` |
   | `metric` | 业务指标 | `key`、`label`、`value`、`unit` |
   | `succeeded` | 成功结束 | `percent`、`message`、`result` |
   | `failed` | 失败结束 | `message`、`result` |

   示例：

   ```json
   {"type":"event","task_id":"L8...","event":"started","message":"开始执行"}
   {"type":"event","task_id":"L8...","event":"step","step":"login","label":"登录","status":"running"}
   {"type":"event","task_id":"L8...","event":"progress","percent":20,"current":12,"total":60,"unit":"节","message":"正在观看"}
   {"type":"event","task_id":"L8...","event":"log","level":"warn","message":"触发限流"}
   {"type":"event","task_id":"L8...","event":"metric","key":"correct","label":"正确率","value":"96","unit":"%"}
   {"type":"event","task_id":"L8...","event":"succeeded","percent":100,"message":"done","result":{"score":"96"}}
   ```

   - `status` ∈ `running | succeeded | failed | skipped`。
   - `level` ∈ `debug | info | warn | error`。
   - 只给 `current/total` 时，Host 自动计算百分比。
   - 未提供 `percent` 的 `step/log/metric` 不会改变任务进度。
   - `result` 可以是对象或数组；数组配合 `actions[].result.type=table` 渲染表格。
   - 取消：Host 会先向 stdin 写一行 `{"type":"cancel"}`，给脚本约 3 秒清理时间，再终止进程。Python SDK 暴露 `ctx.canceled` 与 `ctx.check_canceled()`。
3. 脚本输出终止事件（`succeeded`/`failed`）后即可退出；Host 也会在读尽后结束。
4. 脚本可在**执行前**输出一行 `{"type":"ready","platform":"welearn","version":"0.2.0"}`（可选）。

约束：
- stdout **只能**输出协议 JSON（一行一个）；调试信息走 stderr。
- stdin/stdout 一律使用 **UTF-8**；Host 会为 Python 注入 `PYTHONUTF8=1`、`PYTHONIOENCODING=utf-8`，SDK 也会强制 UTF-8 并以 UTF-8 字节写出。
- 非协议输出会被忽略，并在任务日志中给出一次警告。
- 凭据只来自请求，禁止落盘/打印。
- 若进程 EOF 前没有终止事件，Host 判定为 `failed`。
- Python 作者直接使用 `ctx.progress/progress_of/step/log/metric/result`，见 `docs/script-development.md`。

## 4. 代理与运行时检测

凭据配置了账号级代理时，后端在任务/发现请求中携带 `proxy`，Host 会以环境变量 `OMNITASK_PROXY` 注入脚本进程。Python SDK 的 `omnitask_sdk.http.new_session()` 会自动使用它；自建 HTTP 客户端的脚本也应读取该变量。

每个任务会复制一份脚本源码到独立工作区，并设置：

- `OMNITASK_WORKDIR`：当前任务工作区
- `OMNITASK_SCRIPT_DIR`：原脚本版本目录，只读参考
- `TMPDIR` / `TEMP` / `TMP`、`HOME`、`XDG_CACHE_HOME` 指向工作区
- `OMNITASK_RATE_DB`：Host 级 SQLite 共享限速库，SDK `RateLimiter` 会跨进程协调平台请求间隔

脚本应把 `config.ini`、Cookie、缓存、临时文件写到当前工作区（通常直接用相对路径即可），不要修改原脚本目录。

- Host 加载 manifest 后用 `exec.LookPath(runtime)` 预检；缺失则该脚本标记为不可用、不对外暴露，并在日志中体现。
- Host 在 `Health`/`Execute` 时**热扫描**脚本目录，安装后无需重启。
- 例：缺少 Python 时会提示安装 Python（Host 日志；前端脚本页状态显示“缺少运行时”）。

## 5. 与后端 gRPC 的映射

Host 实现既有 `AgentService`：

- `Health` → 返回可用脚本的 `platforms`。
- `ListScripts` → 返回 Host 实际加载的完整 manifest、hash、运行时与可用性；后端以此同步注册表，修改 manifest 无需重装。
- `Execute(ExecuteRequest)` → 启动对应脚本子进程，把 stdio 事件映射为 `ExecuteEvent` 流返回后端。
- `Discover(DiscoverRequest)` → 列出账号在平台上的条目（课程/任务），支持 `params` 筛选。

proto 的 `ExecuteEvent` 在 v2 增加 `has_percent`、`step/label/status`、`level`、`current/total/unit`、`key/value`、`stage`、`result_json`。后端原样落 `task_events` 并通过 WebSocket 推送；任务详情页可通过 `GET /api/v1/tasks/:id/events` 回放历史事件。

## 6. Python 便捷运行时（SDK）

Python 作者用 `@action` + `serve()` 接入，通过 `ctx` 调用组件：

```python
from omnitask_sdk import action, serve

@action("time")
def time(ctx):
    ctx.step("login", "登录", "running")
    ctx.progress_of(12, 60, "正在挂时长", unit="分钟")
    ctx.metric("watched", "已观看", 12, "分钟")
    ctx.result["watched"] = 60

if __name__ == "__main__":
    serve()
```

完整组件说明与示例见 `docs/script-development.md`。

Host 启动 Python 脚本时会注入 `OMNITASK_SDK=<sdk_dir>`、`PYTHONPATH` 与 `OMNITASK_BANK_ADDR`。

## 7. 独立单跑（CLI/GUI）

脚本入口应同时支持人工使用，例如：

```bash
python main.py --cli --action time --param cid=2416 --credential username=u --credential password=p
```

由脚本作者决定 CLI/GUI 实现；被 Host 调用时走 stdio（无参数）。

## 8. 安全与免责

- 脚本以宿主权限运行，恶意脚本可窃取凭据/攻击主机；仅安装可信来源并固定 commit。
- 平台不背书脚本内容，责任由脚本作者与安装者承担。


## 9. 发现（Discover / @listing）

除 `Execute` 外，Agent 还提供 `Discover`：列出账号在平台上的条目（课程/任务），供前端选择。

- 请求：`DiscoverRequest{platform, params, credentials}`；Host 以子进程发送 `{"type":"discover",...}`。平台页的「筛选」会把参数传入 `params`，带筛选的查询不读写缓存。
- 脚本（Python SDK）：用 `@listing()` 注册发现函数，返回条目列表；运行时归一化后输出
  `{"type":"items","items":[{"key":"...","fields":{...}}]}`。
- manifest 用 `discovery` 声明展示 schema（不新增平台也无需改前端）：
  ```json
  "discovery": {
    "label": "课程",
    "primary": "name",
    "columns": [
      {"key": "name", "label": "课程名", "type": "text", "important": true},
      {"key": "cid", "label": "课程ID", "type": "text"},
      {"key": "per", "label": "进度", "type": "progress", "important": true},
      {"key": "study_time", "label": "学习时长", "type": "text", "important": true, "sort": "duration"}
    ],
    "actions": ["time", "progress"]
  }
  ```
- 后端缓存每个用户在平台的条目，并按 `discovery.min_interval`（默认 60s）限流刷新。
- manifest 顶层可选 `"icon": "mdi-xxx"`，用于前端侧栏/平台页图标；缺省用 `mdi-puzzle-outline` 兜底。
- `discovery.primary` 可选，指定卡片视图的主字段（列 key），缺省用第一列。
- `columns[].important` 可选：任一列标记 `true` 时，表格只展示这些重要列，其余列收进「详细信息」对话框；都不标记时沿用旧行为展示全部列。卡片视图始终展示全部列。
- `columns[].sort` 可选：`number`、`percent`、`duration`、`date`、`text`，前端用它决定表头排序规则；缺省按 `type` 和文本自然排序。不同脚本的字段格式不同，建议为时长/百分比/编号显式标注。
- `columns[].type` 建议取值（前端据此渲染，值仍是字符串）：
  - `text`（默认）、`number`、`date`
  - `progress`：值为 `60%` 或 `0.6` → 进度条/卡片进度环
  - `chip`：值为 `"已完成:success"`（冒号后为可选颜色）→ 彩色标签
  - `sparkline`：值为逗号分隔数字（如 `"1,2,3,5,4"`）→ 迷你折线图
- 平台页表格行提供「动作」菜单（列出 `discovery.actions`）和「详细信息」图标按钮（tooltip + 弹窗展示全部字段）；卡片视图直接展示全部字段与动作。未列出的动作不显示。
