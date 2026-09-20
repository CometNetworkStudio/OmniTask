# 任务参数（用户可自定义）

> 参数的**唯一事实来源是各脚本的 `script.json` manifest**（`params` / `credentials`）。
> 前端按 manifest 动态生成表单；后端原样透传给 Host → 脚本。
> 本文仅列出官方 `cidaren` / `welearn` 脚本的参数作为参考。
> 实现：`sdk/python/core/params.py`（脚本内默认值 < `.env` 基础设施 < 任务 params）。

## 词达人 cidaren

| 参数 | 默认 | 说明 |
|------|------|------|
| `task_type` | `1` | 1 班级学习任务 / 2 班级测试任务 |
| `task_name` | 空 | 仅执行名称匹配的任务（子串） |
| `progress_lt` | `100` | 只处理进度低于该值的任务 |
| `limit` | `0` | 最多执行任务数，0=不限 |
| `max_steps` | `1000` | 单任务最多推进题数（安全上限） |
| `think_min` / `think_max` | `2` / `4` | 每题作答后的“思考”停顿秒数 |
| `spend_min` / `spend_max` | `5` / `15` | 服务端 `time_spent`，单位 500=1 秒 |

## WeLearn welearn

| 参数 | 默认 | 说明 |
|------|------|------|
| `cid` | 第一门课 | 课程 id |
| `unit_idx` | 全部单元 | 只处理该**数字下标**的单元（0-based） |
| `duration` | `60` | 挂时长秒数（`action=time`） |
| `accuracy` | `100` | 提交进度正确率（`action=progress`） |
| `include_completed` | `false` | 是否也处理已完成 SCO |
| `limit` | `0` | 最多处理 SCO 数，0=不限 |

## 通用（两平台）

| 参数 | 默认 | 说明 |
|------|------|------|
| `platform_concurrency` | `2` | 平台并发上限（同账号始终串行） |
| `request_min_interval` / `request_max_interval` | `0.8` / `2.5` | 请求间隔随机区间（秒），风控节流 |

## 凭据（不经 `.env`，M2 起由后端从数据库解密下发）

- 词达人：`token`
- WeLearn：`username` / `password`
