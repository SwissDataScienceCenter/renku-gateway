package config

import (
	"fmt"
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
	RunningEnvironment
	Server     ServerConfig
	Session    SessionConfig
	Revproxy   RevproxyConfig
	Login      LoginConfig
	Redis      RedisConfig
	Monitoring MonitoringConfig
}

type RunningEnvironment string

const Development RunningEnvironment = "development"
const Production RunningEnvironment = "production"

// type DBAdapter interface {
// 	models.AccessTokenGetter
// 	models.AccessTokenSetter
// 	models.AccessTokenRemover
// 	models.RefreshTokenGetter
// 	models.RefreshTokenSetter
// 	models.RefreshTokenRemover
// 	models.SessionGetter
// 	models.SessionSetter
// 	models.SessionRemover
// }

const DBTypeRedis string = "redis"
const DBTypeRedisMock string = "redis-mock"

// var CLISessionCookieOpt = models.WithCookieTemplate(http.Cookie{Name: "_renku_cli_session", Secure: true, HttpOnly: true, Path: "/"})
// var UISessionCookieOpt = models.WithCookieTemplate(http.Cookie{Name: "_renku_ui_session", Secure: true, HttpOnly: true, Path: "/"})

func (c *Config) Validate() error {
	err := c.Session.Validate()
	if err != nil {
		return err
	}
	err = c.Login.Validate(c.RunningEnvironment)
	if err != nil {
		return err
	}
	err = c.Revproxy.Validate()
	if err != nil {
		return err
	}
	return nil
}
