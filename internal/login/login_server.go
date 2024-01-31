package login

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/labstack/echo/v4"
)

type LoginServer struct {
	sessionStore   models.SessionStore
	providerStore  oidc.ClientStore
	tokenStore     models.TokenStore
	sessionHandler models.SessionHandler
	cliSessionHandler models.SessionHandler
	config         *config.LoginConfig
}

func (l *LoginServer) RegisterHandlers(server *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	e := server.Group(l.config.EndpointsBasePath)
	e.Use(commonMiddlewares...)

	wrapper := ServerInterfaceWrapper{Handler: l}
	e.GET(
		"/callback",
		wrapper.GetCallback,
		NoCaching,
	)
	e.GET(
		"/health",
		wrapper.GetHealth,
	)
	e.GET(
		"/login",
		wrapper.GetLogin,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.GET(
		"/logout",
		wrapper.GetLogout,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.POST(
		"/logout",
		wrapper.PostLogout,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.GET(
		"/device/login",
		wrapper.GetDeviceLogin,
		NoCaching,
	)
	e.POST(
		"/device/login",
		wrapper.PostDeviceLogin,
		NoCaching,
	)
	e.POST(
		"/device/logout",
		wrapper.PostDeviceLogout,
		NoCaching,
		l.cliSessionHandler.Middleware(),
	)
	tokenProxyMiddlewares, err := l.DeviceTokenProxy()
	if err != nil {
		slog.Error("LOGIN SERVER INITIALIZATION", "error", err)	
		os.Exit(1)
	}
	// /device/token is just proxied - it does not need a handler on this server
	e.Group("/device/token", tokenProxyMiddlewares...)
}

type LoginServerOption func(*LoginServer) error

func WithConfig(loginConfig config.LoginConfig) LoginServerOption {
	return func(l *LoginServer) error {
		l.config = &loginConfig
		providerStore, err := oidc.NewClientStore(loginConfig.Providers)
		if err != nil {
			return err
		}
		l.providerStore = providerStore
		return nil
	}
}

func WithDBConfig(dbConfig config.RedisConfig) LoginServerOption {
	return func(l *LoginServer) error {
		options := []db.RedisAdapterOption{db.WithRedisConfig(dbConfig)}
		if l.config.TokenEncryption.Enabled && l.config.TokenEncryption.SecretKey != "" {
			options = append(options, db.WithEcryption(string(l.config.TokenEncryption.SecretKey)))
		}
		rdb, err := db.NewRedisAdapter(options...)
		if err != nil {
			return err
		}
		l.tokenStore = &rdb
		l.sessionStore = &rdb
		return nil
	}
}

func WithTokenStore(store models.TokenStore) LoginServerOption {
	return func(l *LoginServer) error {
		l.tokenStore = store 
		return nil
	}
}

func WithSessionStore(store models.SessionStore) LoginServerOption {
	return func(l *LoginServer) error {
		l.sessionStore = store 
		return nil
	}
}

// NewLoginServer creates a new LoginServer that handles the callbacks from oauth2
// and initiates the login flow for users.
func NewLoginServer(options ...LoginServerOption) (*LoginServer, error) {
	server := LoginServer{}
	// by default we setup all dummy storage which in production is overriden later by the options
	dummyStore := db.NewMockRedisAdapter()
	server.tokenStore = &dummyStore
	server.sessionStore = &dummyStore
	server.sessionHandler = models.NewSessionHandler(
		models.WithSessionStore(dummyStore),
		models.WithTokenStore(dummyStore),
	)
	providerStore, err := oidc.NewClientStore(map[string]config.OIDCClient{})
	if err != nil {
		return nil, err
	}
	server.providerStore = providerStore
	for _, opt := range options {
		err := opt(&server)
		if err != nil {
			return nil, err
		}
	}
	sessionHandler := models.NewSessionHandler(
		models.WithSessionStore(server.sessionStore),
		models.WithTokenStore(server.tokenStore),
	)
	cliSessionHandler := models.NewSessionHandler(
		models.WithSessionStore(server.sessionStore),
		models.WithTokenStore(server.tokenStore),
		models.DontCreateIfMissing(),
		models.DontRecreateIfExpired(),
		models.WithCookieTemplate(http.Cookie{
			Name:     models.CliSessionCookieName,
			Secure:   false,
			HttpOnly: true,
			Path:     "/",
			MaxAge:   3600,
		}),
		models.WithHeaderKey(models.CliSessionHeaderKey),
		models.WithContextKey(models.CliSessionCtxKey),
	)
	server.sessionHandler = sessionHandler
	server.cliSessionHandler = cliSessionHandler
	return &server, nil
}
