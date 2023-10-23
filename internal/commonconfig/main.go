// package commonconfig contains common configurations used by different packages
package commonconfig

import (
	"net/http"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

const AnonymousUserHeaderPrefix string = "anon-"
const AnonymousUserHeaderKey string = "Renku-Auth-Anon-Id"
const SessionCookieName string = "_renku_session"
const SessionIDCtxKey string = "_renku_session_id"
const SessionCtxKey string = "_renku_session"
const SessionPersistnceTypeRedis string = "redis"
const SessionPersistnceTypeMock string = "redis-mock"

func DefaultSessionCookieConfig() SessionCookieConfig {
	return SessionCookieConfig{
		Name:     SessionCookieName,
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
	}
}

func SessionTTL() map[models.SessionType]time.Duration {
	return map[models.SessionType]time.Duration{
		models.Default: time.Hour * 8,
		models.Cli:     time.Hour * 48,
	}
}

type RedisConfig struct {
	Addresses  []string `mapstructure:"redis_addresses"`
	IsSentinel bool     `mapstructure:"redis_is_sentinel"`
	Password   string   `mapstructure:"redis_password"`
	MasterName string   `mapstructure:"redis_master_name"`
	DBIndex    int      `mapstructure:"redis_db_index"`
}

type SessionCookieConfig struct {
	Name          string        `mapstructure:"session_cookie_name"`
	Path          string        `mapstructure:"session_cookie_path"`
	Domain        string        `mapstructure:"session_cookie_domain"`
	MaxAgeSeconds int           `mapstructure:"session_cookie_max_age_seconds"`
	Secure        bool          `mapstructure:"session_cookie_secure"`
	HTTPOnly      bool          `mapstructure:"session_cookie_http_only"`
	SameSite      http.SameSite `mapstructure:"session_cookie_same_site"`
}
