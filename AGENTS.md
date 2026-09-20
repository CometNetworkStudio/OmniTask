# OmniTask 项目规则

> 本文件供 AI 助手（opencode 等）读取，必须严格遵守。

## 前端 UI（必须遵守）

1. 只用 Vuetify 官方组件；布局优先 Props / Utility Classes（`pa-*`、`ma-*`、`ga-*`、`d-flex`、`flex-*`）。
2. 禁止为间距/Flex/Grid/颜色/圆角/字体写 CSS；仅 Vuetify 无法实现时才允许新增 CSS。
3. 优先 `v-container`、`v-row`、`v-col`、`v-card`、`v-data-table`、`v-dialog`、`v-form`、`v-toolbar`、`v-navigation-drawer`、`v-list`。
4. Less is More，避免过度动画、渐变、毛玻璃、复杂装饰。
5. 颜色/字体/圆角/阴影走 Vuetify Theme，禁止硬编码。
6. API 走 `api/service` 层，页面禁止 `axios`；状态用 Pinia。
7. **MD3 圆角固定写死**：按钮/列表/表格/alert/card 等统一 `variant="elevated"` + `rounded="xl"`，card 额外 `elevation="2"`；本就圆角的组件（如 `v-chip`）不改。以 Vuetify `defaults` 全局设定，个别组件不支持该 prop 则就近设置。
8. **含用户输入的对话框必须加 `persistent`**。
9. **禁止 `v-snackbar`**，统一用 `v-snackbar-queue`（右上角、多条、bouncy 动画 `transition="bouncy-slide-auto"`）。

## 通用约定

- 前端：Vue3 + JavaScript（禁止 TypeScript）+ Vite。
- 后端：Go + Gin + GORM，规划见 `docs/development-plan.md`；支持 `lite`(SQLite/无 Redis) 与 `full`(MySQL/Redis) 双模式。
- 脚本：平台脚本**不在本仓库**，作为独立项目经 Host 以子进程 + stdio JSON 运行；协议见 `docs/script-protocol.md`，接入见 `docs/integration-guide.md`。Host 为 Go（`cmd/agent`）。
- 密钥/凭据永不入库明文、永不提交到仓库（lite 的 `config.yaml` 除外，它由用户本机生成且 gitignore）。
- 重要决策与进展回写 `docs/context.md` / `docs/development-plan.md`。
- 每次对话结束用 `./deploy.sh` 重新部署到 `https://<your-host>:4096/omnitask/` 供查看。
