# OmniTask Backend

Go + Gin + GORM + gRPC 后端：用户/凭据/任务/题库/脚本注册表与任务编排。

## 运行模式（`config.yaml` 的 `mode`）

- **lite（默认）**：SQLite（`github.com/glebarez/sqlite`，纯 Go 无需 CGO）+ 无 Redis（进度走进程内）。
- **full**：MySQL + Redis。

`config.yaml` 首次运行自动生成（含随机密钥，明文存于其中）；仅当路径为默认 `config.yaml` 时自动创建。数据库迁移：

- lite：启动时 GORM AutoMigrate。
- full：`go run ./cmd/server -migrate -conf config.yaml`（golang-migrate，SQL 见 `internal/migration/migrations`）。

## 组件

| 入口 | 说明 |
|------|------|
| `cmd/server` | HTTP API（前缀 `/api/v1`）+ WebSocket + 任务编排 + `QuestionBank` gRPC；在 `/` 托管内嵌前端 dist；`-migrate` 执行数据库迁移（实现见 `internal/migration`）|
| `cmd/host` | Go 脚本 Host：扫描脚本目录，以子进程 + stdio JSON 运行平台脚本 |

## 目录

```
internal/
├── scripthost/     # Go Host（manifest 解析 / 子进程 stdio / 热扫描）
├── handler/       # HTTP handler（auth/credential/task/question/script/ws）
├── middleware/    # JWT / Casbin 鉴权 / 日志 / CORS
├── model/         # GORM 模型
├── repository/    # 数据访问（驱动可切：sqlite/mysql）
├── service/       # 业务（含编排 Orchestrator、脚本安装、进度）
├── ws/            # WebSocket Hub
└── server/        # HTTP / gRPC / 迁移装配
pkg/               # config / crypto(AES-GCM) / authz(Casbin) / jwt / sid / log ...
```

## 验收

```bash
go build ./... && go vet ./... && go test ./pkg/...
```
