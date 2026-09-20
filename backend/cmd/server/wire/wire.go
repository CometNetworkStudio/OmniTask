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
	"backend/pkg/crypto"
	"backend/pkg/jwt"
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
	repository.NewTaskRepository,
	repository.NewQuestionRepository,
)

var serviceSet = wire.NewSet(
	sid.NewSid,
	jwt.NewJwt,
	crypto.FromEnv,
	authz.NewEnforcer,
	ws.NewHub,
	service.NewAuthService,
	service.NewCredentialService,
	service.NewTaskService,
	service.NewQuestionService,
	service.NewAgentClient,
	service.NewProgressStore,
	service.NewOrchestrator,
)

var handlerSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewCredentialHandler,
	handler.NewTaskHandler,
	handler.NewQuestionHandler,
	handler.NewWSHandler,
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
