//go:build wireinject
// +build wireinject

package wire

import (
	"backend/internal/server"
	"backend/pkg/app"
	"backend/pkg/log"
	"github.com/google/wire"
	"github.com/spf13/viper"
)

// build App
func newApp(
	task *server.TaskServer,
) *app.App {
	return app.NewApp(
		app.WithServer(task),
		app.WithName("backend-task"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		server.NewTaskServer,
		newApp,
	))
}
