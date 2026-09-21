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
)
