# OmniTask Frontend

Vue3 + JavaScript + Vite + Vuetify（MD3）+ Pinia + Vue Router + Axios。

## 开发

```bash
npm install
npm run dev        # http://127.0.0.1:5173，/v1 代理到后端 :8080（含 WebSocket）
```

## 构建（子路径部署）

```bash
VITE_BASE=/omnitask/ npm run build   # 产物 dist/，base 由 VITE_BASE 决定
```

- API/WS 基址基于 `import.meta.env.BASE_URL`，无需改代码即可换部署路径。
- UI 规范必须遵守根目录 `AGENTS.md`（Vuetify 组件、MD3 圆角、`persistent` 弹窗、`v-snackbar-queue`、页面不直接用 axios、状态用 Pinia）。
- API 统一经 `src/api/services/*`；状态在 `src/stores/*`。
