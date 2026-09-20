# 构建与运行说明

## 1. 三个可执行程序

OmniTask 后端编译后会生成三个程序（可按需只编译其中一个）：

| 程序 | 源码入口 | 作用 |
|------|----------|------|
| **omnitask-server** | `backend/cmd/server/main.go` | 主服务：HTTP API（前缀 `/api/v1`）+ WebSocket + 任务编排 + `QuestionBank` gRPC，并在 `/` 托管内嵌的前端 SPA |
| **omnitask-agent** | `backend/cmd/agent/main.go` | 脚本 Host：扫描脚本目录，按 `script.json` 以**子进程 + stdio JSON** 运行平台脚本（实现在 `backend/internal/agenthost`），对后端暴露 gRPC `AgentService` |
| **omnitask-migrate** | `backend/cmd/migrate/main.go` | 数据库迁移（golang-migrate，SQL 在 `backend/cmd/migrate/migrations`），**full 模式**使用 |

> 注意：产物里的 `omnitask-agent-windows-amd64.exe` 就是上表的 Host，编译自 **`backend/cmd/agent`** 目录，入口是 **`backend/cmd/agent/main.go`**；它**不是**仓库根目录的 `sdk/python/`（那是 Python 脚本 SDK）。

## 2. 工作流程（简化）

```
浏览器 ──HTTP /api/v1 + WebSocket──▶ omnitask-server
                                        │ 轮询任务、编排、鉴权、脚本安装
                                        │ gRPC AgentService.Execute(流式)
                                        ▼
                                    omnitask-agent (Host)
                                        │ 子进程 + stdio JSON
                                        ▼
                                    平台脚本(任意语言) ──gRPC QuestionBank──▶ server
```

- `omnitask-server` 负责用户与数据；不直接执行脚本。
- `omnitask-agent` 只负责“加载/运行脚本”，可用单独机器部署（未来多机）。
- `omnitask-migrate` 只在部署（full 模式）时执行一次建表/升级。
- 前端 dist 通过 `go:embed` 打进 `omnitask-server`，实现**单文件**（浏览器访问 server 的 `/`）。

## 3. 构建参数

构建脚本 `omnitask.sh` / `omnitask.bat` 使用：

```
CGO_ENABLED=0 GOOS=<linux|windows> GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w" -o <输出> ./cmd/<入口>
```

- `CGO_ENABLED=0`：纯静态、无需 C 工具链，便于交叉编译（依赖全为纯 Go：`glebarez/sqlite`、`go-sql-driver/mysql`、`casbin`）。
- `-trimpath`：去掉源码绝对路径，产物可复现、更小。
- `-ldflags "-s -w"`：去掉符号表（`-s`）和 DWARF 调试信息（`-w`），**功能不变**，体积明显减小。
- `GOOS/GOARCH`：交叉编译目标（本脚本统一 `amd64`）；Windows 产物带 `.exe`。
- 产物命名区分平台：`omnitask-server-linux-amd64`、`omnitask-server-windows-amd64.exe`。

## 4. 两种运行模式

- **lite（默认）**：SQLite + 无 Redis。直接运行 `omnitask-server` → 当前目录自动生成 `config.yaml`（含随机密钥）→ 浏览器打开 `http://127.0.0.1:8080`，首个注册用户为管理员。
- **full**：MySQL + Redis。先 `omnitask-migrate` 迁移，再以指向 MySQL/Redis 的配置启动 `server` 与 `agent`；两者也可与前端一起由服务器统一托管。

## 5. 一键构建

```bash
./omnitask.sh            # 交互菜单
./omnitask.sh all both   # 非交互：前端 + server/agent/migrate，linux+windows amd64
```

产物输出到 `build/`（已 gitignore）。
