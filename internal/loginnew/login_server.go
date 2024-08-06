package loginnew

import (
	"fmt"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/login"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

type LoginServer2 struct {
	config         *config.LoginConfig
	providerStore  oidc.ClientStore
	sessionHandler *sessions.SessionHandler
}

func (l *LoginServer2) RegisterHandlers(server *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	e := server.Group(l.config.EndpointsBasePath)
	e.Use(commonMiddlewares...)

	wrapper := login.ServerInterfaceWrapper{Handler: l}
	e.GET(
		"/login",
		wrapper.GetLogin,
		login.NoCaching,
	)
	e.GET(
		"/callback",
		wrapper.GetCallback,
		login.NoCaching,
	)
}

type LoginServer2Option func(*LoginServer2) error

func WithConfig(loginConfig config.LoginConfig) LoginServer2Option {
	return func(l *LoginServer2) error {
		l.config = &loginConfig
		providerStore, err := oidc.NewClientStore(loginConfig.Providers)
		if err != nil {
			return err
		}
		l.providerStore = providerStore
		return nil
	}
}

func WithSessionHandler(sh *sessions.SessionHandler) LoginServer2Option {
	return func(l *LoginServer2) error {
		l.sessionHandler = sh
		return nil
	}
}

// NewLoginServer creates a new LoginServer that handles the callbacks from oauth2
// and initiates the login flow for users.
func NewLoginServer(options ...LoginServer2Option) (LoginServer2, error) {
	server := LoginServer2{}
	for _, opt := range options {
		err := opt(&server)
		if err != nil {
			return LoginServer2{}, err
		}
	}
	if server.config == nil {
		return LoginServer2{}, fmt.Errorf("login server config not provided")
	}
	if server.providerStore == nil {
		return LoginServer2{}, fmt.Errorf("OIDC providers not initialized")
	}
	if server.sessionHandler == nil {
		return LoginServer2{}, fmt.Errorf("session handler not initialized")
	}
	return server, nil
}
