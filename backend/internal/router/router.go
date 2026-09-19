package router

import (
	"backend/internal/handler"
	"backend/pkg/jwt"
	"backend/pkg/log"
	"github.com/spf13/viper"
)

type RouterDeps struct {
	Logger            *log.Logger
	Config            *viper.Viper
	JWT               *jwt.JWT
	AuthHandler       *handler.AuthHandler
	CredentialHandler *handler.CredentialHandler
	TaskHandler       *handler.TaskHandler
	QuestionHandler   *handler.QuestionHandler
}
