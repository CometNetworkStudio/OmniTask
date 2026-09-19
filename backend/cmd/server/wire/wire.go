//go:build wireinject
// +build wireinject

package wire

import (
	"backend/internal/router"
	"backend/internal/server"
	"backend/pkg/app"
	"backend/pkg/jwt"
	"backend/pkg/log"
	"backend/pkg/server/http"
	"github.com/google/wire"
	"github.com/spf13/viper"
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
		wire.Struct(new(router.RouterDeps), "*"),
		server.NewHTTPServer,
		server.NewJobServer,
		jwt.NewJwt,
		newApp,
	))
}
