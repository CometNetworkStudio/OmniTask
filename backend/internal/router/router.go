package router

import (
	"backend/pkg/jwt"
	"backend/pkg/log"
	"github.com/spf13/viper"
)

type RouterDeps struct {
	Logger *log.Logger
	Config *viper.Viper
	JWT    *jwt.JWT
}
