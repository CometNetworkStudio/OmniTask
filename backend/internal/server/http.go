package server

import (
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
	s := http.NewServer(
		gin.Default(),
		deps.Logger,
		http.WithServerHost(deps.Config.GetString("http.host")),
		http.WithServerPort(deps.Config.GetInt("http.port")),
	)

	// swagger doc
	docs.SwaggerInfo.BasePath = "/v1"
	s.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		//ginSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", deps.Config.GetInt("app.http.port"))),
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.PersistAuthorization(true),
	))

	s.Use(
		middleware.CORSMiddleware(),
		middleware.ResponseLogMiddleware(deps.Logger),
		middleware.RequestLogMiddleware(deps.Logger),
		//middleware.SignMiddleware(log),
	)
	s.GET("/", func(ctx *gin.Context) {
		apiV1.HandleSuccess(ctx, map[string]interface{}{
			"name":    "OmniTask",
			"version": "0.1.0",
		})
	})

	v1Group := s.Group("/v1")
	{
		auth := v1Group.Group("/auth")
		auth.POST("/register", deps.AuthHandler.Register)
		auth.POST("/login", deps.AuthHandler.Login)

		authed := v1Group.Group("")
		authed.Use(middleware.StrictAuth(deps.JWT, deps.Logger))
		authed.POST("/credentials", deps.CredentialHandler.Upsert)
		authed.GET("/credentials", deps.CredentialHandler.List)
		authed.POST("/tasks", deps.TaskHandler.Create)
		authed.GET("/tasks", deps.TaskHandler.List)
		authed.GET("/tasks/:id", deps.TaskHandler.Get)
		authed.GET("/questions", deps.QuestionHandler.List)
		authed.POST("/questions", deps.QuestionHandler.Save)
		authed.GET("/questions/lookup", deps.QuestionHandler.Lookup)
	}

	return s
}
