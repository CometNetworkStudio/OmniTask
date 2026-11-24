package server

import (
	"context"
	"net"
	"strings"
	"time"

	apiV1 "backend/api/v1"
	"backend/docs"
	"backend/internal/middleware"
	"backend/internal/router"
	"backend/pkg/server/http"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func NewHTTPServer(
	deps router.RouterDeps,
) *http.Server {
	// 启动时按已安装脚本补齐平台默认配置。
	deps.PlatformConfig.Bootstrap(context.Background())

	if deps.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	opts := []http.Option{
		http.WithServerHost(deps.Config.GetString("http.host")),
		http.WithServerPort(deps.Config.GetInt("http.port")),
	}
	tlsEnabled := deps.Config.GetBool("http.tls.enabled")
	if tlsEnabled {
		opts = append(opts, http.WithServerTLS(
			true,
			deps.Config.GetString("http.tls.cert_file"),
			deps.Config.GetString("http.tls.key_file"),
		))
	} else if !isLoopbackHost(deps.Config.GetString("http.host")) {
		deps.Logger.Warn("HTTP 未启用 TLS 且监听非回环地址；令牌与凭据可能明文传输。请由反向代理终结 TLS 并限制网络访问。",
			zap.String("host", deps.Config.GetString("http.host")))
	}
	// 不用 gin.Default()：其访问日志会打印含 accessToken 的原始查询串，
	// 这里改用自定义的脱敏日志中间件，仅保留 panic 恢复。
	s := http.NewServer(gin.New(), deps.Logger, opts...)

	// 只信任显式配置的反向代理，避免客户端伪造 X-Forwarded-For 绕过限流。
	proxies := make([]string, 0)
	for _, proxy := range strings.Split(deps.Config.GetString("http.trusted_proxies"), ",") {
		if trimmed := strings.TrimSpace(proxy); trimmed != "" {
			proxies = append(proxies, trimmed)
		}
	}
	if err := s.SetTrustedProxies(proxies); err != nil {
		deps.Logger.Warn("设置可信代理失败，将不信任任何代理", zap.Error(err))
	}

	// swagger doc（生产默认关闭，可用 http.swagger 强制开启）
	if deps.Config.GetBool("http.swagger") || deps.Config.GetString("env") != "prod" {
		docs.SwaggerInfo.BasePath = "/api/v1"
		s.GET("/swagger/*any", ginSwagger.WrapHandler(
			swaggerfiles.Handler,
			ginSwagger.DefaultModelsExpandDepth(-1),
			ginSwagger.PersistAuthorization(true),
		))
	}

	origins := make([]string, 0)
	for _, origin := range strings.Split(deps.Config.GetString("http.cors_origins"), ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	s.Use(
		gin.Recovery(),
		middleware.SecurityHeaders(),
		middleware.CORSMiddleware(origins),
		middleware.ResponseLogMiddleware(deps.Logger),
		middleware.RequestLogMiddleware(deps.Logger),
		//middleware.SignMiddleware(log),
	)
	s.GET("/api/health", func(ctx *gin.Context) {
		apiV1.HandleSuccess(ctx, map[string]interface{}{
			"name":    "OmniTask",
			"version": "0.1.0",
		})
	})

	v1Group := s.Group("/api/v1")
	{
		auth := v1Group.Group("/auth")
		auth.Use(middleware.IPRateLimit(deps.Config.GetInt("security.auth_rate_limit_per_minute"), time.Minute))
		auth.POST("/register", deps.AuthHandler.Register)
		auth.POST("/login", deps.AuthHandler.Login)

		authed := v1Group.Group("")
		authed.Use(
			middleware.StrictAuth(deps.JWT, deps.Logger),
			middleware.Authorize(deps.Casbin, deps.Logger),
		)
		authed.POST("/credentials", deps.CredentialHandler.Create)
		authed.PUT("/credentials/:id", deps.CredentialHandler.Update)
		authed.DELETE("/credentials/:id", deps.CredentialHandler.Archive)
		authed.GET("/credentials", deps.CredentialHandler.List)
		authed.GET("/credentials/access-logs", deps.CredentialHandler.AccessLogs)
		authed.POST("/tasks", deps.TaskHandler.Create)
		authed.GET("/tasks", deps.TaskHandler.List)
		authed.GET("/tasks/:id", deps.TaskHandler.Get)
		authed.GET("/tasks/:id/events", deps.TaskHandler.Events)
		authed.POST("/tasks/:id/cancel", deps.TaskHandler.Cancel)
		authed.POST("/tasks/:id/retry", deps.TaskHandler.Retry)
		authed.DELETE("/tasks/:id", deps.TaskHandler.Delete)
		authed.GET("/questions", deps.QuestionHandler.List)
		authed.POST("/questions", deps.QuestionHandler.Save)
		authed.GET("/questions/lookup", deps.QuestionHandler.Lookup)

		authed.GET("/scripts", deps.ScriptHandler.List)
		authed.POST("/scripts/install", deps.ScriptHandler.Install)
		authed.GET("/scripts/install/:job", deps.ScriptHandler.InstallJob)
		authed.GET("/scripts/:id", deps.ScriptHandler.Get)
		authed.POST("/scripts/:id/reinstall", deps.ScriptHandler.Reinstall)
		authed.POST("/scripts/:id/rollback", deps.ScriptHandler.Rollback)
		authed.POST("/scripts/:id/debug", deps.ScriptHandler.Debug)
		authed.POST("/scripts/:id/enable", deps.ScriptHandler.Toggle)
		authed.DELETE("/scripts/:id", deps.ScriptHandler.Uninstall)

		authed.GET("/me", deps.UserHandler.Me)
		authed.GET("/users", deps.UserHandler.List)
		authed.POST("/users/:id/role", deps.UserHandler.UpdateRole)
		authed.POST("/users/:id/enable", deps.UserHandler.SetEnabled)

		authed.GET("/platforms", deps.PlatformHandler.List)
		authed.GET("/platforms/:id/items", deps.PlatformHandler.Items)
		authed.POST("/platforms/:id/discover", deps.PlatformHandler.Refresh)

		ws := v1Group.Group("/ws")
		ws.Use(middleware.NoStrictAuth(deps.JWT, deps.Logger))
		ws.GET("", deps.WSHandler.Handle)
	}

	registerStatic(s, deps.Logger)

	return s
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
