//go:build wireinject
// +build wireinject

package wire

import (
	"backend/internal/handler"
	"backend/internal/repository"
	"backend/internal/router"
	"backend/internal/server"
	"backend/internal/service"
	"backend/pkg/app"
	"backend/pkg/crypto"
	"backend/pkg/jwt"
	"backend/pkg/log"
	"backend/pkg/server/http"
	"backend/pkg/sid"
	"github.com/google/wire"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
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
	service.NewAuthService,
	service.NewCredentialService,
	service.NewTaskService,
	service.NewQuestionService,
)

var handlerSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewCredentialHandler,
	handler.NewTaskHandler,
	handler.NewQuestionHandler,
)

// build App
func newApp(
	httpServer *http.Server,
	jobServer *server.JobServer,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, jobServer),
		app.WithName("backend"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serviceSet,
		handlerSet,
		wire.Struct(new(router.RouterDeps), "*"),
		server.NewHTTPServer,
		server.NewJobServer,
		newApp,
	))
}
