# OmniTask 脚本开发与接入指南

OmniTask 的平台能力全部来自**可插拔脚本**。本文档面向想为新平台开发脚本的开发者。

> 一句话：写一个 Python 脚本，暴露 `run(action, params, credentials)` 生成器 + 一份 `script.json`，即可被 OmniTask 安装调用；同时它也能单独用 CLI 运行。

---

## 1. 概念

- **脚本（Script）= 平台（Platform）**：一个脚本对应一个学习平台。
- 脚本**不属于主仓库**，独立开源发布（避免版权与依赖耦合），主仓库通过"安装"引入。
- 主仓库与脚本之间只有两个契约：**manifest** 与 **gRPC/运行时接口**。

## 2. 脚本仓库结构

```
omnitask-script-example/
├── script.json          # manifest（必须）
├── runner.py            # 入口，实现 run()
├── client.py            # 平台协议（建议拆分）
├── cli.py               # 独立命令行（建议）
├── requirements.txt
└── README.md
```

## 3. manifest：`script.json`

```json
{
  "id": "example",
  "name": "示例平台",
  "version": "0.1.0",
  "author": "your-github-id",
  "entry": "runner.py",
  "requires": ["requests", "brotli"],
  "credentials": [
    { "key": "token", "label": "Token", "type": "password" }
  ],
  "actions": [
    { "id": "study", "name": "学习任务" },
    { "id": "time", "name": "挂时长" }
  ],
  "params": [
    { "key": "duration", "label": "时长(秒)", "type": "number", "default": 60 },
    { "key": "limit", "label": "最多数量(0=不限)", "type": "number", "default": 0 }
  ]
}
```

- `id`：平台唯一标识，入库与前端路由用。
- `credentials`：凭据字段；后端加密存储，仅在任务下发时解密并随 gRPC 传入。
- `actions`：动作列表；`run()` 的 `action` 取值。
- `params`：参数 schema，前端据此生成表单；`type` ∈ `text|number|switch`。

## 4. runner 契约

> 完整协议（Host 与脚本的 stdio NDJSON 约定）见 **`docs/script-protocol.md`**。
> 脚本由 Go Host 以子进程启动：Host 写一行 execute 请求到 stdin，脚本逐行输出事件 JSON 到 stdout。

Python 作者最简方式：实现生成器 + 一行 `serve`。

```python
# runner.py
def run(action: str, params: dict[str, str], credentials: dict[str, str]):
    """逐条 yield 进度事件；失败直接 raise。"""
    yield {"percent": 0, "message": "开始", "result": {}}
    ...
    yield {"percent": 100, "message": "完成", "result": {"ok": "1"}}
```

```python
# main.py
import importlib, os, sys
_HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.environ.get("OMNITASK_SDK") or os.path.dirname(os.path.dirname(_HERE)))
sys.path.insert(0, os.path.dirname(_HERE))
from runtime import serve
run = importlib.import_module(f"{os.path.basename(_HERE)}.runner").run
serve(run)
```

事件约定：`{"percent": int, "message": str, "result": dict|None}`。异常会被转成 `failed` 事件。

**运行时提供的能力**（无需自己实现网络层）：

- `from core.http import new_session, get_json, post_json`：带重试/超时的会话。
- `from core.ratelimit import RateLimiter`：平台并发 + 账号串行 + 随机间隔。
- `from core.bank import GrpcBank`：经后端读写题库（Agent 不直连数据库）。
- `core.config.Config`：读取 env 配置。

## 5. gRPC 契约

后端 → 脚本：

```proto
service AgentService {
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc Execute(ExecuteRequest) returns (stream ExecuteEvent);
}
message ExecuteRequest {
  string task_id = 1;
  string platform = 2;
  string action = 3;
  map<string, string> params = 4;
  map<string, string> credentials = 5;  // 后端解密后下发，脚本不得持久化
}
enum EventType { EVENT_TYPE_STARTED=1; EVENT_TYPE_PROGRESS=2; EVENT_TYPE_LOG=3; EVENT_TYPE_SUCCEEDED=4; EVENT_TYPE_FAILED=5; }
```

脚本 → 后端（题库）：

```proto
service QuestionBank {
  rpc Lookup(LookupRequest) returns (LookupResponse);
  rpc Save(SaveRequest) returns (SaveResponse);
}
```

完整定义见 `proto/agent/v1/agent.proto`。

## 6. 参数与凭据

- **参数**由后端用户自定义并透传（`map<string,string>`），脚本负责类型转换与默认值。
- **凭据**示例：`{"token": "..."}` 或 `{"username": "...", "password": "..."}`。
- 严禁把凭据写入日志或落盘。

## 7. CLI（建议但强烈推荐）

脚本应同时提供独立 CLI，便于本地调试与脱离平台使用：

```bash
python cli.py --action study --param course_id=XX --credential token=YY
```

## 8. 本地自测

```bash
pip install -r requirements.txt
python cli.py --action <action> --credential <k=v> --param <k=v>
```

- 纯函数（签名、解码、策略）写单测。
- 网络请求做少量真实/脱敏验证。

## 9. 发布与接入

官方脚本（已开源，可作为模板）：
- 词达人：https://github.com/CometNetworkStudio/cidaren
- WeLearn：https://github.com/CometNetworkStudio/welearn

接入步骤：

1. 在自己仓库按上述规范实现（`script.json` + `main.py`）并保证默认分支可下载。
2. 管理员的 OmniTask「脚本」页填写仓库地址（或 zip 地址），可选加速源（如 `https://gh-proxy.com/`，前端会拼接）。
3. 后端下载 zip → 解压到运行目录 `scripts/<id>/` → 校验 `runtime` → 写入注册表（状态：可用 / 缺少运行时）。Host **热扫描**，无需重启。
4. 之后即可在任务页选择该平台创建任务。

> 当前后端按仓库**默认分支**下载（`/archive/refs/heads/main.zip`）；指定 commit/tag 的固定安装为后续增强。
>
> 安全：仅**管理员**可安装脚本；安装前请审阅源码。脚本以宿主权限运行，恶意脚本可窃取凭据或攻击主机，请只安装可信来源。

## 10. Vibecoding Prompt（可直接复制给 AI）

```text
你是 Python 自动化脚本开发者，请为「<平台名>」开发一个适配 OmniTask 的脚本。

背景：OmniTask 是校园任务自动化平台，Agent 通过 gRPC 调用脚本。运行时已提供：
- core.http（requests 会话，带重试/超时/Brotli）
- core.ratelimit（RateLimiter：平台并发+账号串行+随机间隔）
- core.bank.GrpcBank（经后端读写题库，绝不直连数据库）
- core.config.Config（读 env）

要求：
1. 产出目录 scripts/<platform_id>/：runner.py（入口）、client.py（协议）、script.json（manifest）、cli.py（本地命令行）、requirements.txt。
2. runner.py 必须实现生成器：
   def run(action, params, credentials):
       yield {"percent": int, "message": str, "result": dict|None}
   失败直接 raise。
3. script.json 按规范填写 id/name/version/actions/params/credentials。
4. 参数从 params 读取并做类型转换与默认值；凭据只从 credentials 读取，禁止落盘/打印。
5. 请求节流用 core.ratelimit，禁止硬编码 sleep 风暴；标识平台风控错误并抛出可读异常。
6. 需要题目答案时用 GrpcBank.Lookup/Save 读写题库；只用平台返回的权威结果回写。
7. cli.py 支持 --action / --param k=v / --credential k=v，便于脱离平台调试。
8. 为签名/解码/答题策略等纯函数写单测。
9. 不要复制任何 GPL/无协议项目的源码；只依据平台网络协议独立实现，MIT 开源。

请先给出实现计划与文件清单，再逐个文件输出完整代码。
```

## 11. 免责声明

脚本仅供学习研究，使用者需遵守目标平台服务条款；脚本作者对其实现与法律合规负责。
