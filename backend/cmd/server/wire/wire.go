//go:build wireinject
// +build wireinject

package wire

import (
	"backend/internal/handler"
	"backend/internal/repository"
	"backend/internal/router"
	"backend/internal/server"
	"backend/internal/service"
	"backend/internal/ws"
	"backend/pkg/app"
	"backend/pkg/authz"
	"backend/pkg/config"
	"backend/pkg/crypto"
	"backend/pkg/log"
	grpcserver "backend/pkg/server/grpc"
	"backend/pkg/server/http"
	"backend/pkg/sid"
	"github.com/google/wire"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	repository.NewRedis,
	repository.NewRepository,
	repository.NewUserRepository,
	repository.NewCredentialRepository,
	repository.NewCredentialAccessLogRepository,
	repository.NewPlatformConfigRepository,
	repository.NewTaskRepository,
	repository.NewQuestionRepository,
	repository.NewScriptRepository,
	repository.NewScriptInstallJobRepository,
	repository.NewPlatformCacheRepository,
	repository.NewSessionRepository,
	repository.NewLoginAttemptRepository,
)

var serviceSet = wire.NewSet(
	sid.NewSid,
	config.NewStore,
	crypto.FromConfig,
	authz.NewEnforcer,
	ws.NewHub,
	service.NewSessionService,
	service.NewAuthService,
	service.NewPlatformConfigService,
	service.NewCredentialService,
	service.NewTaskService,
	service.NewQuestionService,
	service.NewScriptService,
	service.NewDiscoverService,
	service.NewUserService,
	service.NewAgentClient,
	service.NewProgressStore,
	service.NewOrchestrator,
)

var handlerSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewCredentialHandler,
	handler.NewTaskHandler,
	handler.NewQuestionHandler,
	handler.NewScriptHandler,
	handler.NewPlatformHandler,
	handler.NewUserHandler,
	handler.NewWSHandler,
	handler.NewConfigHandler,
)

// build App
func newApp(
	httpServer *http.Server,
	jobServer *server.JobServer,
	bankServer *grpcserver.Server,
	orchestrator *service.Orchestrator,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, jobServer, bankServer, orchestrator),
		app.WithName("backend"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serviceSet,
		handlerSet,
		server.NewBankGRPCServer,
		wire.Struct(new(router.RouterDeps), "*"),
		server.NewHTTPServer,
		server.NewJobServer,
		newApp,
	))
}
