// package common contains common configurations used by different packages
package common

import (
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

const SessionCtxKey = "_renku_session"

var CLISessionCookieOpt = models.WithCookieTemplate(http.Cookie{Name: "_renku_cli_session", Secure: true, HttpOnly: true, Path: "/"})
var UISessionCookieOpt = models.WithCookieTemplate(http.Cookie{Name: "_renku_ui_session", Secure: true, HttpOnly: true, Path: "/"})

type RedisConfig struct {
	Addresses  []string `mapstructure:"redis_addresses"`
	IsSentinel bool     `mapstructure:"redis_is_sentinel"`
	Password   string   `mapstructure:"redis_password"`
	MasterName string   `mapstructure:"redis_master_name"`
	DBIndex    int      `mapstructure:"redis_db_index"`
}
