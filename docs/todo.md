# OmniTask 待办（Backlog）

> 产品方向：**小范围多用户**。P0 已在本轮完成（见下），本文只保留未完成项。
> **当本文所有条目标记完成时，可清空/删除本文内容。**
> 进度总览见 `docs/context.md` 第 20 节。

## 已完成（P0，本轮）
- 任务生命周期：取消 / 重试 / 删除接口 + 前端按钮；server 重启把 `running/queued` 标记为 `interrupted`；支持配置 `agent.task_timeout` 超时。
- 注册控制：`security.registration: open|closed`。
- 生产默认关闭 Swagger（`http.swagger`）。
- CORS 收敛为可配置白名单（`http.cors_origins`，默认同源）。
- 自检：`./omnitask.sh check` / `omnitask.bat` 菜单 + GitHub Actions（`.github/workflows/ci.yml`）。

## P0 补充
- [ ] 启用 GitHub Actions：把 `docs/github-actions-ci.yml` 复制到 `.github/workflows/ci.yml` 提交（需 token 具备 `workflow` 权限，或在 GitHub 网页添加）。

## P1 体验 / 功能
### 已完成
- [x] `/api/v1/me`：返回当前用户与角色；前端启动 `fetchMe` 刷新。
- [x] 用户管理（admin）：`GET /users`、改角色、启用/禁用；前端「用户」页。
- [x] 分页：后端默认/上限由 `pagination.default|max` 控制；前端封装 `components/DataTable.vue`（MD3、自研分页页脚，替换 v-data-table 默认页脚）。
- [x] 布局：抽屉分组导航 + 底部账号/退出/版权；顶栏头像不再被裁剪。

### 待做
- [ ] 脚本页：**安装进度条**（当前为按钮 loading，非真实进度）。
- [ ] 脚本安装的依赖自动安装（隔离 venv）。

### 本轮已完成（P1 剩余）
- [x] 任务列表按状态/平台筛选（后端查询参数 + 前端下拉）。
- [x] 脚本安装支持指定分支/标签/commit（`ref`）；「运行时缺失」帮助弹窗。
- [x] backend / agent **systemd 服务化**（`deploy/systemd/`，服务器重启自动拉起）。

## P2 生态 / 长期
- [ ] 脚本依赖自动安装（隔离 venv）或更明确的安装提示。
- [ ] SDK 多语言示例（Node/Go 各一个最小实现）。
- [ ] 新平台脚本（学习通、知到等）。
- [ ] 多机 Agent（已评估，暂不做；需要时再加 agent_id/心跳）。
- [ ] 脚本安装固定 commit/tag + 签名/白名单（供应链安全）。

## 工程卫生 / 运维
- [ ] 删除 Nunu 遗留：`backend/cmd/task`、`backend/scripts/`。
- [ ] 补充后端 service/handler 测试（当前仅 `pkg/crypto` 有测试）。
- [ ] lite(AutoMigrate) 与 full(SQL 迁移) 双机制的 schema 漂移检查。
- [ ] 日志轮转/清理策略核对；健康探针。
- [ ] 凭据主密钥轮换方案（当前明文存 config.yaml）。

## 运营项（非代码）
- [ ] 词达人 token 触发 `11003` 风控，需在 App/微信完成安全验证后重试。
