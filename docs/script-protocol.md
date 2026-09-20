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

```
<scripts_dir>/
└── <script-id>/
    ├── script.json     # 必须
    ├── main.py         # entry（示例）
    └── ...
```

`script.json`：

```json
{
  "id": "welearn",
  "name": "WeLearn",
  "version": "0.1.0",
  "author": "CometNetworkStudio",
  "runtime": "python3",
  "entry": "main.py",
  "args": [],
  "requires": ["requests"],
  "actions": [
    { "id": "time", "name": "挂时长" },
    { "id": "progress", "name": "提交进度" }
  ],
  "credentials": [
    { "key": "username", "label": "账号", "type": "text" },
    { "key": "password", "label": "密码", "type": "password" }
  ],
  "params": [
    { "key": "cid", "label": "课程 id", "type": "text", "default": "" },
    { "key": "duration", "label": "挂时长(秒)", "type": "number", "default": 60 }
  ]
}
```

- `runtime`：可执行名（`python3`、`node`、或脚本自带二进制路径）。
- `entry`：相对脚本目录的入口；Host 以 `runtime entry args...` 启动，`cwd` 为脚本目录。
- `requires`：运行时依赖说明，Host 只用于展示/预检提示（不自动安装）。

## 3. 运行协议（NDJSON）

Host 启动脚本后，**一个任务一个子进程**：

1. Host 向脚本 **stdin** 写一行请求 JSON：
   ```json
   {"type":"execute","task_id":"L8...","platform":"welearn","action":"time",
    "params":{"cid":"2416","duration":"60"},
    "credentials":{"username":"...","password":"..."},
    "bank_addr":"127.0.0.1:9000"}
   ```
2. 脚本向 **stdout** 逐行输出事件 JSON：
   ```json
   {"type":"event","task_id":"L8...","event":"progress","percent":10,"message":"登录成功","result":{}}
   {"type":"event","task_id":"L8...","event":"succeeded","percent":100,"message":"done","result":{}}
   ```
   - `event` ∈ `progress | succeeded | failed`。
   - 出错输出 `{"type":"event","event":"failed","message":"...","result":{}}`。
3. 脚本输出终止事件（`succeeded`/`failed`）后即可退出；Host 也会在读尽后结束。
4. 脚本可在**执行前**输出一行 `{"type":"ready","platform":"welearn","version":"0.1.0"}`（可选）。

约束：
- stdout **只能**输出协议 JSON（一行一个）；调试信息走 stderr。
- 凭据只来自请求，禁止落盘/打印。
- 若进程 EOF 前没有终止事件，Host 判定为 `failed`。

## 4. 运行时检测

- Host 加载 manifest 后用 `exec.LookPath(runtime)` 预检；缺失则该脚本标记为不可用、不对外暴露，并在日志与 `Health` 中体现。
- 例：缺少 Python 时提示用户安装 Python（Host 日志 + 前端脚本页状态，后续实现）。

## 5. 与后端 gRPC 的映射

Host 实现既有 `AgentService`：

- `Health` → 返回可用脚本的 `platforms`。
- `Execute(ExecuteRequest)` → 启动对应脚本子进程，把 stdio 事件映射为 `ExecuteEvent` 流返回后端。

后端**无需任何改动**。

## 6. Python 便捷运行时（SDK）

为降低 Python 作者门槛，提供 `runtime.serve(run_func)`：作者只需实现与现在相同的生成器

```python
def run(action, params, credentials):
    yield {"percent": 10, "message": "..."}
```

`main.py`：

```python
import os, sys
sys.path.insert(0, os.environ.get("OMNITASK_SDK", ""))
from runtime import serve
from runner import run

if __name__ == "__main__":
    serve(run)
```

Host 启动 Python 脚本时会注入 `OMNITASK_SDK=<sdk_dir>` 与 `OMNITASK_BANK_ADDR`。

## 7. 独立单跑（CLI/GUI）

脚本入口应同时支持人工使用，例如：

```bash
python main.py --cli --action time --param cid=2416 --credential username=u --credential password=p
```

由脚本作者决定 CLI/GUI 实现；被 Host 调用时走 stdio（无参数）。

## 8. 安全与免责

- 脚本以宿主权限运行，恶意脚本可窃取凭据/攻击主机；仅安装可信来源并固定 commit。
- 平台不背书脚本内容，责任由脚本作者与安装者承担。
