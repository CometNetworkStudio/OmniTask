# 部署模式：自托管与官方托管

OmniTask 用一个代码仓库同时服务两类部署。差异靠配置和边缘网关实现，不维护两个 fork。

## 自托管

适合个人、家庭或小团队。默认配置即可运行：

- `security.registration: open`，首个注册用户自动成为管理员
- 不启用邀请码、验证码、邮箱验证
- 登录失败锁定默认 5 次 / 15 分钟，可在配置里设为 0 关闭
- 任务配额默认宽松
- 会话 Cookie 使用 `HttpOnly`、`SameSite=Lax`，`Secure` 根据请求自动判断

如果部署在公网，至少做三件事：

1. 前面加反向代理并启用 HTTPS
2. 配置 `http.trusted_proxies`，否则限流会把所有人算到代理 IP
3. 先创建管理员，再决定是否保持开放注册

## 官方托管

在同一份代码上使用更严格的配置，并把机器人防护放在边缘层：

```yaml
env: prod

http:
  trusted_proxies: 172.16.0.0/12
  cors_origins: ""

security:
  registration: invite
  invite_code: "替换为随机邀请码"
  allow_first_user_admin: false
  auth_rate_limit_per_minute: 10
  session_ttl_hours: 168
  cookie_secure: auto
  cookie_same_site: lax
  login_max_failures: 5
  login_window_minutes: 15

task:
  max_active_per_user: 2
  daily_tasks_per_user: 20
  max_pending_per_user: 5
  max_pending: 500
```

管理员通过环境变量预创建，避免公开部署时依赖“第一个注册者”：

```bash
OMNITASK_ADMIN_USER=admin
OMNITASK_ADMIN_PASSWORD='至少 8 位的强密码'
```

## 边缘层建议

应用只负责会话、锁定和配额。批量注册、撞库、扫描和 DDoS 交给网关：

- Cloudflare Turnstile 或 Managed Challenge 放在 `/api/v1/auth/register`
- nginx `limit_req` 限制 `/api/v1/auth/*` 和任务创建
- WAF 拦截异常 UA、空 Origin、已知恶意 ASN
- 隐藏源站，只允许边缘节点回源
- 对注册量、失败登录、任务创建量做告警

这些配置放在部署环境，不进入开源代码，自托管用户不需要关心。

## 鉴权与升级说明

- 认证方式为服务端会话 Cookie，不再使用 JWT。
- Cookie 只保存随机令牌，数据库保存 SHA-256 摘要。
- 禁用账号、修改角色、吊销会话都会在下次请求立即生效。
- 从 JWT 版本升级后，旧登录状态全部失效，用户需要重新登录。
- 非安全方法需要携带 `X-CSRF-Token`，值来自 `omnitask_csrf` Cookie。
- WebSocket 同样使用会话 Cookie，并要求任务归属或管理员权限。

## 协议与合规

- 应用内公开页面：`/terms`、`/privacy`，未登录可访问；设置页「协议」展示同一版本。
- 门户：`https://omnitasks.top/legal.html`，包含用户协议、隐私政策、数据保留与删除规则、滥用投诉和封禁流程。
- 注册必须勾选同意，后端记录协议版本与同意时间。
- 滥用和隐私请求加入 QQ 频道 `https://pd.qq.com/s/N834h64oek`（频道号 `pd84113918`），敏感信息不要发到公开 Issue。
- 修改协议时同步更新后端 `CurrentTermsVersion`、前端 `frontend/src/content/legal.js` 和门户 `legal.html`。

## 管理员设置

管理员在「设置」页会看到额外的「站点安全」「站点配置」标签：

- 站点安全：查看注册方式、会话有效期、登录锁定、任务配额，吊销任意会话，查看最近登录尝试
- 站点配置：可视化修改注册方式、邀请码、登录防护、会话、任务配额和发现间隔
- 可视化修改写入主配置同目录的 `*.overrides.yml`，不覆盖主配置注释
- 保存后安全子集立即生效；监听地址、数据库、Agent、日志和密钥需要重启
- 修改主配置或覆盖文件后，可点「重新读取文件」立即刷新可编辑字段

普通用户可在「设置 / 账号安全」查看自己的登录设备并退出其他设备。
