package login

import (
	"fmt"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

type LoginServer struct {
	config        *config.LoginConfig
	providerStore oidc.ClientStore
	sessions      *sessions.SessionStore
	tokenStore    models.TokenStoreInterface
}

func (l *LoginServer) RegisterHandlers(server *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	e := server.Group(l.config.LoginRoutesBasePath)
	e.Use(commonMiddlewares...)

	wrapper := ServerInterfaceWrapper{Handler: l}
	e.GET("/login", wrapper.GetLogin, NoCaching)
	e.GET("/callback", wrapper.GetCallback, NoCaching)
	e.GET("/logout", wrapper.GetLogout, NoCaching)
	e.GET("/user-profile", wrapper.GetUserProfile)
	e.GET("/gitlab/exchange", l.GetGitLabToken, NoCaching)
	e.GET("/gitlab/logout", l.GetGitLabLogout)
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

func WithSessionStore(sessions *sessions.SessionStore) LoginServerOption {
	return func(l *LoginServer) error {
		l.sessions = sessions
		return nil
	}
}

func WithTokenStore(store models.TokenStoreInterface) LoginServerOption {
	return func(l *LoginServer) error {
		l.tokenStore = store
		return nil
	}
}

// NewLoginServer creates a new LoginServer that handles the callbacks from oauth2
// and initiates the login flow for users.
func NewLoginServer(options ...LoginServerOption) (*LoginServer, error) {
	server := LoginServer{}
	for _, opt := range options {
		err := opt(&server)
		if err != nil {
			return &LoginServer{}, err
		}
	}
	if server.config == nil {
		return &LoginServer{}, fmt.Errorf("login server config not provided")
	}
	if server.providerStore == nil {
		return &LoginServer{}, fmt.Errorf("OIDC providers not initialized")
	}
	if server.sessions == nil {
		return &LoginServer{}, fmt.Errorf("session store not initialized")
	}
	if server.tokenStore == nil {
		return &LoginServer{}, fmt.Errorf("token store is not initialized")
	}
	return &server, nil
}
