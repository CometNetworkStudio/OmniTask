# OmniTask Script SDK (Python)

OmniTask 平台脚本的 Python SDK。提供网络会话、限流、题库客户端、配置读取，以及 **stdio JSON 运行时**，
让脚本既能被 OmniTask 的 Go Host 以子进程调用，也能独立 CLI 运行。

协议见仓库根 `docs/script-protocol.md`；接入总览见 `docs/integration-guide.md`。

## 目录

| 路径 | 说明 |
|------|------|
| `runtime.py` | `serve(run_func)`：读写 stdin/stdout 的 stdio JSON 协议实现 |
| `core/http.py` | requests 会话（重试/超时/Brotli） |
| `core/ratelimit.py` | 平台并发 + 同账号串行 + 随机间隔 |
| `core/bank.py` | 题库抽象：`LocalBank`(SQLite) / `GrpcBank`(经后端 gRPC) |
| `core/config.py` | 读取环境变量（可选 `.env`） |
| `core/llm.py` | OpenAI 兼容 LLM 客户端（可选） |
| `core/params.py` | 参数默认值与合并 |
| `core/events.py` | 进度事件约定 |
| `gen/agent/v1/` | 由 `make proto` 生成的 gRPC 桩（题库客户端） |
| `tests/` | SDK 单元测试 |

## 在自己的脚本项目中使用

约定：脚本仓库里实现 `runner.py` 的 `run(action, params, credentials)` 生成器，
再加一个 `main.py` 调 `serve(run)`（`script.json` 见协议文档）。

**方式 A：拷贝（推荐，简单、无依赖安装）**

把本目录的 `core/`、`runtime.py`、`gen/` 拷进你的脚本仓库根目录，然后：

```python
# runner.py
def run(action, params, credentials):
    yield {"percent": 10, "message": "开始", "result": {}}
    ...
```

```python
# main.py
import importlib, os, sys
_HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _HERE)                       # core/ runtime.py gen/ 与 main.py 同级
from runtime import serve
run = importlib.import_module(f"{os.path.basename(_HERE)}.runner").run
serve(run)
```

**方式 B：PYTHONPATH 指向本目录**（不拷贝，开发期方便）

```bash
PYTHONPATH=/path/to/OmniTask/sdk/python python your_script/main.py
```

## 常用能力

```python
from core.http import new_session, get_json, post_json
from core.ratelimit import RateLimiter
from core.bank import GrpcBank, LocalBank
from core.config import Config

cfg = Config.from_env()                      # 读取环境变量/可选 .env
session = new_session(headers={"User-Agent": "..."})
data = get_json(session, "https://example.com/api")

# 题库：被 OmniTask 调用时用 GrpcBank（Host 会注入 OMNITASK_BANK_ADDR / OMNITASK_BANK_MODE）
bank = GrpcBank(cfg.bank_addr) if cfg.bank_mode == "grpc" else LocalBank(cfg.bank_local_path)
hit = bank.lookup("yourplatform", "question-key")
bank.save({"platform": "yourplatform", "question_key": "...", "answer": "..."}, overwrite=True)
```

进度事件：`yield {"percent": int, "message": str, "result": dict|None}`；异常直接 `raise`，
运行时会把异常转成 `failed` 事件（详见 `docs/script-protocol.md`）。

## 依赖

```
pip install grpcio protobuf requests brotli
```

## 测试

```bash
cd sdk/python && ./.venv/bin/python -m unittest discover -s tests -t .
```
