# OmniTask Script SDK (Python)

让平台脚本快速接入 OmniTask：提供 stdio JSON 运行时、动作注册、网络/限流/题库等能力，
**不规定文件名、不要求改造既有项目结构**。

协议：`docs/script-protocol.md` ｜ 接入总览：`docs/integration-guide.md`

## 安装

```bash
# 从本仓库子目录安装（无需 PyPI）
pip install "git+https://github.com/CometNetworkStudio/OmniTask.git#subdirectory=sdk/python"
```

依赖：`grpcio`、`protobuf`、`requests`、`brotli`、`PySocks`。

## 三种接入方式

### A. 装饰器（推荐新脚本）

```python
from omnitask_sdk import action, serve

@action("study")
def study(ctx):
    ctx.step("login", "登录", "running")
    ctx.progress(10, "登录中")
    ctx.step("login", "登录", "succeeded")

    ctx.progress_of(12, 144, "正在观看", unit="节")
    ctx.metric("correct", "正确率", 96, "%")
    ctx.log("已完成 12 节", level="info")

    ctx.result["score"] = 100

if __name__ == "__main__":
    serve()
```

`script.json`：`{"protocol":2,"id":"x","name":"X","runtime":"python3","entry":"<你的文件>.py","actions":[{"id":"study","name":"学习"}]}`

### B. 生成器（兼容旧写法）

```python
from omnitask_sdk import serve

def run(action, params, credentials):
    yield {"percent": 10, "message": "..."}

serve(run)
```

### C. CLI 桥（零代码接入已有命令行项目）

在脚本仓库**只写 `script.json`**，把自己的命令填进 action：

```json
{
  "id": "my-cli", "name": "某 CLI 项目", "runtime": "python3",
  "module": "omnitask_sdk.bridge",
  "actions": [
    { "id": "run", "name": "运行", "command": ["mycli", "run", "--course", "{cid}"] }
  ],
  "credentials": [{ "key": "token", "label": "Token", "type": "password" }],
  "params": [{ "key": "cid", "label": "课程 id", "type": "text", "default": "" }]
}
```

- `module: "omnitask_sdk.bridge"` 表示由 SDK 的桥启动；
- `command` 支持 `{key}` 占位符（取自 params / credentials / action / task_id）；
- 输出行含 `[progress] 42 处理中` 会被识别为进度（可用 `progress_regex` 自定义），其余作为日志；
- 退出码 0 → 成功，非 0 → 失败。

## 能力

| 模块 | 说明 |
|------|------|
| `omnitask_sdk.runtime` | `serve(target=None)`：stdio JSON 运行时 |
| `omnitask_sdk.decorator` | `action(name)` / `registered_actions()` |
| `omnitask_sdk.context` | `AdapterContext`：参数、凭据、组件与结果 |
| `omnitask_sdk.listing` | `listing()`：发现（列出平台条目，供 OmniTask 选择） |
| `omnitask_sdk.bridge` | CLI 桥 |
| `omnitask_sdk.cli` | `check` / `run` / `schema` 开发命令 |
| `omnitask_sdk.http` | requests 会话（重试/超时/Brotli/代理，自动读取 `OMNITASK_PROXY`） |
| `omnitask_sdk.ratelimit` | 平台并发 + 同账号串行 + 随机间隔；自动读取 `OMNITASK_RATE_DB` 做跨进程节流 |
| `omnitask_sdk.bank` | 题库：`LocalBank`(SQLite) / `GrpcBank`(经后端) |
| `omnitask_sdk.config` | 读取环境变量（可选 `.env`） |
| `omnitask_sdk.llm` | OpenAI 兼容 LLM（可选） |
| `omnitask_sdk.gen` | protobuf 生成物（随包分发，题库客户端用） |

## 组件

| 组件 | 方法 |
|---|---|
| 进度 | `ctx.progress(percent, message)` |
| 数量进度 | `ctx.progress_of(current, total, message, unit="")` |
| 步骤 | `ctx.step(key, label, status)` |
| 日志 | `ctx.log(message, level="info")`，另有 `debug/warn/error` |
| 指标 | `ctx.metric(key, label, value, unit="")` |
| 结果 | `ctx.result["key"] = value`，或直接赋值数组配合 table 视图 |
| 取消 | `ctx.canceled` / `ctx.check_canceled()` |

完整说明见 `docs/script-development.md`。

## 接入注意

- 每个任务在独立工作区运行，`config.ini`、Cookie、缓存请写相对路径或 `OMNITASK_WORKDIR`，不要写脚本安装目录。
- `.venv` 不复制到工作区；子进程请用 `sys.executable`。
- 不要依赖工作区跨任务持久化；成功后默认清理，失败任务默认保留。
- 脚本目录会按任务复制，避免放超大资源文件。
- 同一账号串行，不同账号可并发；使用 `RateLimiter` 可自动读取 `OMNITASK_RATE_DB` 做跨进程节流。
- 代理从 `OMNITASK_PROXY` 读取，`new_session()` 已自动处理。
- 长任务循环中调用 `ctx.check_canceled()`。
- stdout 只能输出协议 JSON，调试信息走 stderr 或 `ctx.log()`。
- `active.json` 和版本目录由平台维护，不要手动修改。

## 开发命令

```bash
python -m omnitask_sdk.cli check
python -m omnitask_sdk.cli run --action study --param cid=2416
python -m omnitask_sdk.cli schema
```

安装后可用 `omnitask-script` 替代 `python -m omnitask_sdk.cli`。

## 测试

```bash
cd sdk/python && ./.venv/bin/python -m unittest discover -s tests -t .
```
