# OmniTask 项目规则

> 本文件供 AI 助手（opencode 等）读取。前端 UI 规则另见
> `frontend/.cursor/rules/Frontend-UI-Rules.mdc`（Cursor 规则），两处保持一致。

## 前端 UI（必须遵守）

1. 使用 Vuetify 官方组件完成页面。
2. 优先使用 Props、Utility Classes（`pa-*`、`ma-*`、`ga-*`、`d-flex`、`flex-*` 等）完成布局。
3. 禁止为了调整间距、Flex、Grid、颜色、圆角、字体等编写 CSS。
4. 只有 Vuetify 无法实现时，才允许新增 CSS。
5. 优先使用 `v-container`、`v-row`、`v-col`、`v-card`、`v-data-table`、`v-dialog`、`v-form`、`v-toolbar`、`v-navigation-drawer`、`v-list` 等组件。
6. 页面保持 Less is More 风格，避免过度动画、渐变、毛玻璃和复杂装饰。
7. 所有颜色、字体、圆角、阴影必须使用 Vuetify Theme，不允许硬编码。
8. API 调用统一通过 `api/service` 层，页面禁止直接使用 `axios`。
9. 状态统一使用 `Pinia`。

## 其它通用约定

- 前端：Vue3 + JavaScript（禁止 TypeScript）+ Vite。
- 后端：Go + Gin + GORM，遵循 `docs/development-plan.md`。
- Agent：Python，平台脚本独立实现（见 `docs/reference-analysis.md`）。
- 密钥/凭据永不入库明文、永不提交到仓库。
- 重要决策与进展回写 `README.md` / `docs/development-plan.md`。
