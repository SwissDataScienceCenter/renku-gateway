package login

import (
	"fmt"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

type LoginServer struct {
	config         *config.LoginConfig
	providerStore  oidc.ClientStore
	sessionHandler *sessions.SessionHandler
}

func (l *LoginServer) RegisterHandlers(server *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	e := server.Group(l.config.EndpointsBasePath)
	e.Use(commonMiddlewares...)

	wrapper := ServerInterfaceWrapper{Handler: l}
	e.GET("/login", wrapper.GetLogin, NoCaching)
	e.GET("/callback", wrapper.GetCallback, NoCaching)
	e.GET("/test", l.GetAuthTest, NoCaching)
}

type LoginServer2Option func(*LoginServer) error

func WithConfig(loginConfig config.LoginConfig) LoginServer2Option {
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

func WithSessionHandler(sh *sessions.SessionHandler) LoginServer2Option {
	return func(l *LoginServer) error {
		l.sessionHandler = sh
		return nil
	}
}

// NewLoginServer creates a new LoginServer that handles the callbacks from oauth2
// and initiates the login flow for users.
func NewLoginServer(options ...LoginServer2Option) (LoginServer, error) {
	server := LoginServer{}
	for _, opt := range options {
		err := opt(&server)
		if err != nil {
			return LoginServer{}, err
		}
	}
	if server.config == nil {
		return LoginServer{}, fmt.Errorf("login server config not provided")
	}
	if server.providerStore == nil {
		return LoginServer{}, fmt.Errorf("OIDC providers not initialized")
	}
	if server.sessionHandler == nil {
		return LoginServer{}, fmt.Errorf("session handler not initialized")
	}
	return server, nil
}
