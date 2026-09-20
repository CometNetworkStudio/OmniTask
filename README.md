# OmniTask 万务通

> 一处托管，万务自达 · 面向大学生的校园任务自动化聚合平台

OmniTask 把重复、耗时的线上学习任务交给自动化脚本执行：用户一次配置，平台编排任务、脚本并发执行，前端实时查看进度，无需守着电脑或手机。

> 开发文档与决策记录：`docs/context.md` ｜ 开发规划：`docs/development-plan.md` ｜ 脚本接入：`docs/integration-guide.md`

---

## 痛点

- 大学生要面对四六级、计算机二级、普通话、学科竞赛，以及学校布置的大量线上任务（学习通时长、词达人、WeLearn 作业/挂时长等），重复且耗时。
- 现有开源脚本零散、要装 Python 环境、要开 GUI、跑起来就被占用一台设备，普通人用不起来。
- 部分人拿这些开源脚本当"信息差"牟利，不公平，而需求真实存在。

## OmniTask 做什么

- **聚合**：一个平台统一管理多个学习平台的自动化任务。
- **零环境**：用户只开浏览器；脚本在服务端无界面执行。
- **可编排**：任务参数高度自定义（程序提供合理默认值）。
- **实时**：WebSocket 实时推送执行进度与结果。
- **可扩展**：脚本以插件形式接入，任何人可按规范开发并发布自己的平台脚本。

## 功能

| 模块 | 说明 |
|------|------|
| 账号体系 | 注册 / 登录（JWT），角色 `admin` / `user` |
| 凭据管理 | 平台凭据 AES-256-GCM 加密入库，仅任务下发瞬间解密 |
| 任务中心 | 按平台/动作创建任务，参数动态表单；状态机与历史 |
| 实时进度 | WebSocket 事件流 + 进度落库 |
| 题库 | 答题结果沉淀，命中直接作答，减少 LLM 调用 |
| 脚本生态 | 插件化平台脚本（manifest + gRPC 契约），管理员安装 |

## 架构

```
浏览器 (Vue3 + Vuetify)
        │ REST + WebSocket
        ▼
Go 后端 (Gin + GORM + MySQL + Redis) ── 编排 / 存储 / 鉴权
        │ gRPC (AgentService.Execute 流式)
        ▼
Python Agent (平台脚本，无界面) ── 实际执行
        ▲
        └── gRPC QuestionBank（脚本经后端读写题库）
```

三层职责严格分离：前端只交互、后端只编排、Agent 只执行（不直连数据库）。

## 技术栈

- **前端**：Vue3 + JavaScript + Vite + Pinia + Vue Router + Axios + Vuetify（MD3）
- **后端**：Go + Gin + GORM + MySQL + Redis + gRPC，迁移用 golang-migrate，权限用 Casbin
- **Agent**：Python 3.12 + gRPC

## 快速开始

```bash
# 1. 起 MySQL / Redis
docker compose -f backend/deploy/docker-compose/docker-compose.yml up -d

# 2. 配置环境变量
cp .env.example .env      # 填 OMNITASK_MASTER_KEY、平台/LLM 等

# 3. 迁移
cd backend && go run ./cmd/migrate -conf config/local.yml

# 4. 起后端（另开终端）
cd backend && go run ./cmd/server -conf config/local.yml

# 5. 起 Agent（另开终端）
cd agent && ./.venv/bin/python grpc_server.py

# 6. 起前端
cd frontend && npm install && npm run dev
```

默认后端 `:8080`、Agent `:50051`、前端 `:5173`。一键部署（本仓库部署环境）见 `./deploy.sh`。

## 脚本生态：开发与接入

OmniTask 的平台能力全部来自**可插拔脚本**。你也想支持一个新平台？按规范写一个带 gRPC/CLI 的 Python 脚本即可被本平台安装调用。

👉 **完整接入与开发文档（含 Vibecoding Prompt）：[docs/integration-guide.md](docs/integration-guide.md)**

## 免责声明

本项目仅供个人学习与技术研究使用。请遵守各平台服务条款与学校相关规定，因使用本项目产生的一切后果由使用者自行承担。本项目不包含任何第三方平台脚本代码，脚本由其作者独立发布并自负责任。

## License

[MIT](LICENSE) © CometNetworkStudio
