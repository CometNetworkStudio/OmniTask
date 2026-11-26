package v1

var (
	// common errors
	ErrSuccess             = newError(0, "ok")
	ErrBadRequest          = newError(400, "Bad Request")
	ErrUnauthorized        = newError(401, "Unauthorized")
	ErrNotFound            = newError(404, "Not Found")
	ErrInternalServerError = newError(500, "Internal Server Error")

	// more biz errors
	ErrEmailAlreadyUse = newError(1001, "The email is already in use.")

	ErrUserAlreadyExists    = newError(1002, "用户名已存在")
	ErrInvalidCredentials   = newError(1003, "用户名或密码错误")
	ErrCredentialNotFound   = newError(1004, "凭据不存在")
	ErrTaskNotFound         = newError(1005, "任务不存在")
	ErrUnsupportedPlatform  = newError(1006, "不支持的平台")
	ErrCredentialEncrypt    = newError(1007, "凭据加密失败")
	ErrCredentialDecrypt    = newError(1008, "凭据解密失败")
	ErrMasterKeyUnavailable = newError(1009, "主密钥未配置")
	ErrForbidden            = newError(1010, "无权限访问")
	ErrTaskNotCancelable    = newError(1011, "当前任务不可取消")
	ErrRegistrationClosed   = newError(1012, "注册已关闭")
	ErrAccountDisabled      = newError(1013, "账号已被禁用")
	ErrCredentialLimit      = newError(1014, "该平台凭据数量已达上限")
	ErrInvalidTaskParams    = newError(1015, "任务参数不合法")
	ErrTaskLimit            = newError(1016, "活跃任务数量已达上限")
	ErrQueueFull            = newError(1017, "任务队列已满，请稍后再试")
	ErrCredentialAudit      = newError(1018, "凭据访问审计失败，已阻止使用")
	ErrTooManyRequests      = newError(1019, "请求过于频繁，请稍后再试")
	ErrInviteInvalid        = newError(1020, "邀请码无效")
	ErrSessionNotFound      = newError(1021, "会话不存在")
	ErrDailyTaskLimit       = newError(1022, "今日任务次数已达上限")
	ErrPendingTaskLimit     = newError(1023, "待处理任务已达上限")
	ErrCSRFInvalid          = newError(1024, "请求校验失败，请刷新页面后重试")
)
