package config

import (
	"fmt"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

type RedactedString string

func (r RedactedString) String() string {
	return fmt.Sprintf("<redacted-%d-chars>", len(r))
}

func (r RedactedString) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", r.String())), nil
}

func (r RedactedString) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r RedactedString) MarshalBinary() ([]byte, error) {
	return []byte(r.String()), nil
}

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

func (c *Config) Validate() error {
	err := c.Login.Validate()
	if err != nil {
		return err
	}
	err = c.Revproxy.Validate()
	if err != nil {
		return err
	}
	return nil
}

