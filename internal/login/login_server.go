package login

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/db"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/labstack/echo/v4"
)

type LoginServer struct {
	sessionStore   SessionStore
	providerStore  oidc.ClientStore
	tokenStore     TokenStore
	sessionHandler models.SessionHandler
	config         *config.LoginConfig
}

func (l *LoginServer) RegisterHandlers(server *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	e := server.Group("/api")
	e.Use(commonMiddlewares...)

	wrapper := ServerInterfaceWrapper{Handler: l}
	e.GET(
		l.config.EndpointsBasePath+"/callback",
		wrapper.GetCallback,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.POST(
		l.config.EndpointsBasePath+"/device/token",
		wrapper.PostDeviceToken,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.POST(
		l.config.EndpointsBasePath+"/device",
		wrapper.PostDevice,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.GET(
		l.config.EndpointsBasePath+"/health",
		wrapper.GetHealth,
	)
	e.GET(
		l.config.EndpointsBasePath+"/login",
		wrapper.GetLogin,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.GET(
		l.config.EndpointsBasePath+"/login/device",
		wrapper.GetDeviceLogin,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.GET(
		l.config.EndpointsBasePath+"/logout",
		wrapper.GetLogout,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
	e.POST(
		l.config.EndpointsBasePath+"/logout",
		wrapper.PostLogout,
		NoCaching,
		l.sessionHandler.Middleware(),
	)
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
	server.sessionHandler = sessionHandler
	return &server, nil
}
