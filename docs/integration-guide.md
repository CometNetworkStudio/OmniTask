# OmniTask 脚本开发与接入指南

OmniTask 的平台能力来自**可插拔脚本**。脚本是**独立项目**，可任意语言；你**不必改文件名、不必改造既有项目结构**。

> 协议细节：`docs/script-protocol.md` ｜ Python SDK：`sdk/python/README.md`

---

## 1. 先选一种接入方式

| 你的情况 | 推荐方式 | 需要写的代码 |
|---|---|---|
| 已有成熟项目（不想动它） | **A. 包装仓** | 一个 `adapter.py` + `script.json` |
| 新写脚本 / 愿意加几行 | **B. 装饰器** | 在已有文件加 `@action` + `serve()` |
| 已有命令行工具 | **C. CLI 桥** | **零代码**，只写 `script.json` |

三种方式最终都由 OmniTask 的 Go Host 以子进程运行（stdio JSON），见协议文档。

## 2. 安装 SDK（方式 A/B 需要）

```bash
pip install "git+https://github.com/CometNetworkStudio/OmniTask.git#subdirectory=sdk/python"
```

## 3. 方式 A：包装仓（原项目一行不改）

新建一个小仓库（或目录）：

```
omnitask-mytool/
├── script.json          # manifest
├── adapter.py           # 把 OmniTask 的 action 映射到你的项目
└── requirements.txt     # 例如 mytool @ git+https://github.com/you/mytool
```

```python
# adapter.py
from omnitask_sdk import action, serve
import mytool                      # 你已发布的库

@action("run")
def run(ctx):
    ctx.progress(5, "启动")
    mytool.do_work(course=ctx.params.get("cid"), token=ctx.credentials["token"])
    ctx.progress(100, "完成")

if __name__ == "__main__":
    serve()
```

`script.json`：`entry` 指向 `adapter.py`（名字任意）。

## 4. 方式 B：装饰器（新脚本）

```python
# 任意文件名，例如 entry.py
from omnitask_sdk import action, serve

@action("study")
def study(ctx):
    ctx.progress(10, "登录成功")
    ...
    ctx.result["score"] = 100

if __name__ == "__main__":
    serve()
```

## 5. 方式 C：CLI 桥（零代码）

脚本仓库**只放 `script.json`**，把你的命令填进去：

```json
{
  "id": "mytool",
  "name": "MyTool",
  "runtime": "python3",
  "module": "omnitask_sdk.bridge",
  "actions": [{ "id": "run", "name": "运行", "command": ["mytool", "run", "--cid", "{cid}"] }],
  "credentials": [{ "key": "token", "label": "Token", "type": "password" }],
  "params": [{ "key": "cid", "label": "课程 id", "type": "text", "default": "" }]
}
```

- `module: omnitask_sdk.bridge` 表示由 SDK 桥启动；
- `{cid}`/`{token}` 等占位符从 params/credentials 取；
- 子进程输出含 `[progress] 42 处理中` → 进度，其余 → 日志；退出码 0=成功。

## 6. manifest（`script.json`）

字段规范见 `docs/script-protocol.md`。要点：
- **入口任意**：`entry`（文件）或 `module`（`python -m`）二选一；**没有 `runner.py` 这种硬性要求**。
- `actions`：动作列表（对应 `run(action)`）。
- `credentials` / `params`：字段 schema，前端据此生成表单。

## 7. 参数、凭据与上下文

- 参数来自用户任务（`ctx.params`）；凭据由后端解密后下发（`ctx.credentials`），**禁止落盘/打印**。
- 凭据字段 `type` 可选：`text`、`password`、`token`、`cookie`。
  - `password` / `token`：前端以密文（遮罩）输入；
  - `cookie`：前端以多行文本粘贴；
  - 若平台同时支持账号密码与令牌/Cookie，**优先声明令牌字段**并引导用户优先使用（短时效、可撤销，泄露后危害更小）。
- 进度：`ctx.progress(percent, message)`；日志：`ctx.log(message)`；结果：`ctx.result[...] = ...`。
- 需要题库时用 `omnitask_sdk.bank.GrpcBank`（Host 会注入 `OMNITASK_BANK_ADDR`/`OMNITASK_BANK_MODE`）。

## 8. 独立运行（CLI）

脚本应能脱离平台单跑，便于调试。SDK 不强制；可用 `argparse` 包一层你自己的入口。

## 9. 发布与接入

1. 在 GitHub 建脚本仓库（含 `script.json`）。
2. 管理员的 OmniTask「脚本」页填仓库地址（可选加速源如 `https://gh-proxy.com/`）。
3. 后端把 zip 下载解压到运行目录 `scripts/<id>/`，Host 热扫描生效。
4. 之后即可在任务页选择该平台。

示例（官方，可作为模板）：
- https://github.com/CometNetworkStudio/cidaren
- https://github.com/CometNetworkStudio/welearn

> 目前按默认分支 `main` 安装；指定 commit/tag 的固定安装为后续增强。

## 10. 安全与免责

- 仅**管理员**可安装脚本；安装前请审阅源码。
- 脚本以宿主权限运行，恶意脚本可窃取凭据/攻击主机，请只安装可信来源。
- 脚本仅供学习研究，使用者需遵守目标平台条款；脚本作者自负责任。
- **凭据信任模型**：服务端在任务运行时必须解密凭据（AES-256-GCM，数据库/备份泄露不会暴露明文），且每次解密都会写入审计记录（「凭据 → 访问记录」可自查）。但任何**代用户登录的服务器在技术上都能拿到明文**——若不信任部署者，请自托管；**不要使用来路不明的公开实例**。
- 建议为各平台使用**独立密码**，避免与邮箱、银行等重要账号相同，以防撞库。

---

## 11. Vibecoding Prompt（复制给 AI）

```text
你是 Python 开发者，请把一个已有项目「<项目名/仓库>」接入 OmniTask 平台。

背景：OmniTask 通过 Go Host 以子进程运行脚本，脚本用 stdio JSON 协议（见 docs/script-protocol.md）。
提供 Python SDK：omnitask_sdk（安装：pip install "git+https://github.com/CometNetworkStudio/OmniTask.git#subdirectory=sdk/python"）。

我的项目情况：<简述，例如“已有 CLI：mytool run --cid X”或“已有函数 lib.start(...)”>

请选择最省事的接入方式并产出：
- 若已有 CLI：只写 script.json，用 "module":"omnitask_sdk.bridge" 与 actions[].command，不要写代码。
- 若已有库：写一个 adapter.py（文件名任意），用 @action("...") 包装，ctx.progress 上报进度，ctx.credentials 取凭据。
- 若新写：用 @action + serve()，入口文件名自定。
要求：
1. script.json 的 id/name/version/actions/params/credentials 完整。
2. 凭据只用 ctx.credentials，禁止打印/落盘。
3. 不要复制任何 GPL/无协议项目源码；独立实现，MIT 开源。
4. 给出目录结构、每个文件完整内容、以及本地如何测试（含 stdio 手动测试示例）。
```
