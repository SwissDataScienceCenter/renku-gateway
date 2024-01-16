package config

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"net/http"
)

type Config struct {
	Server     ServerConfig
	Revproxy   RevproxyConfig
	Login      LoginConfig
	Redis      RedisConfig
	Monitoring MonitoringConfig
}

type DBAdapter interface {
	models.AccessTokenGetter
	models.AccessTokenSetter
	models.AccessTokenRemover
	models.RefreshTokenGetter
	models.RefreshTokenSetter
	models.RefreshTokenRemover
	models.SessionGetter
	models.SessionSetter
	models.SessionRemover
}

const DBTypeRedis string = "redis"
const DBTypeRedisMock string = "redis-mock"

const SessionCtxKey = "_renku_session"

var CLISessionCookieOpt = models.WithCookieTemplate(http.Cookie{Name: "_renku_cli_session", Secure: true, HttpOnly: true, Path: "/"})
var UISessionCookieOpt = models.WithCookieTemplate(http.Cookie{Name: "_renku_ui_session", Secure: true, HttpOnly: true, Path: "/"})
