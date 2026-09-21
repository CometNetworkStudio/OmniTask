package server

import (
	"strings"

	apiV1 "backend/api/v1"
	"backend/docs"
	"backend/internal/middleware"
	"backend/internal/router"
	"backend/pkg/server/http"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewHTTPServer(
	deps router.RouterDeps,
) *http.Server {
	if deps.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	opts := []http.Option{
		http.WithServerHost(deps.Config.GetString("http.host")),
		http.WithServerPort(deps.Config.GetInt("http.port")),
	}
	if deps.Config.GetBool("http.tls.enabled") {
		opts = append(opts, http.WithServerTLS(
			true,
			deps.Config.GetString("http.tls.cert_file"),
			deps.Config.GetString("http.tls.key_file"),
		))
	}
	s := http.NewServer(gin.Default(), deps.Logger, opts...)

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
		auth.POST("/register", deps.AuthHandler.Register)
		auth.POST("/login", deps.AuthHandler.Login)

		authed := v1Group.Group("")
		authed.Use(
			middleware.StrictAuth(deps.JWT, deps.Logger),
			middleware.Authorize(deps.Casbin, deps.Logger),
		)
		authed.POST("/credentials", deps.CredentialHandler.Upsert)
		authed.GET("/credentials", deps.CredentialHandler.List)
		authed.POST("/tasks", deps.TaskHandler.Create)
		authed.GET("/tasks", deps.TaskHandler.List)
		authed.GET("/tasks/:id", deps.TaskHandler.Get)
		authed.POST("/tasks/:id/cancel", deps.TaskHandler.Cancel)
		authed.POST("/tasks/:id/retry", deps.TaskHandler.Retry)
		authed.DELETE("/tasks/:id", deps.TaskHandler.Delete)
		authed.GET("/questions", deps.QuestionHandler.List)
		authed.POST("/questions", deps.QuestionHandler.Save)
		authed.GET("/questions/lookup", deps.QuestionHandler.Lookup)

		authed.GET("/scripts", deps.ScriptHandler.List)
		authed.POST("/scripts/install", deps.ScriptHandler.Install)
		authed.POST("/scripts/:id/enable", deps.ScriptHandler.Toggle)
		authed.DELETE("/scripts/:id", deps.ScriptHandler.Uninstall)

		ws := v1Group.Group("/ws")
		ws.Use(middleware.NoStrictAuth(deps.JWT, deps.Logger))
		ws.GET("", deps.WSHandler.Handle)
	}

	registerStatic(s, deps.Logger)

	return s
}
