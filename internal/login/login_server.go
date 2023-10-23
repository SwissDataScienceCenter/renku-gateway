package main

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/commonconfig"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/commonmiddlewares"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type LoginServer struct {
	sessionStore  SessionStore
	providerStore oidc.ClientStore
	tokenStore    TokenStore
	config        *LoginServerConfig
	echo          *echo.Echo
}

// defaultProviders generates a list of login providers from the providerstore based
// on the default providers specified in the configuration.
func (l *LoginServer) defaultProviders() ([]oidc.Client, bool) {
	output := []oidc.Client{}
	for _, id := range l.config.DefaultProviderIDs {
		provider, found := l.providerStore[id]
		if !found {
			return []oidc.Client{}, false
		}
		output = append(output, provider)
	}
	return output, true
}

// SetProviderStore sets a specific provider store on the login server.
func (l *LoginServer) SetProviderStore(providerStore oidc.ClientStore) {
	l.providerStore = providerStore
}

// NewLoginServer creates a new LoginServer that handles the callbacks from oauth2
// and initiates the login flow for users.
func NewLoginServer(config *LoginServerConfig) (*LoginServer, error) {
	store, err := config.PersistenceAdapter()
	if err != nil {
		return nil, err
	}
	providerStore, err := config.ProviderStore()
	if err != nil {
		return nil, err
	}
	server := &LoginServer{
		sessionStore:  store,
		tokenStore:    store,
		config:        config,
		providerStore: providerStore,
	}

	e := echo.New()
	e.Use(
		middleware.Recover(),
	)
	sessionMiddleware := commonmiddlewares.NewSessionMiddleware(
		store,
		commonconfig.SessionCookieName,
		!config.sessionCookieNotSecure,
	)
	commonMiddleware := []echo.MiddlewareFunc{
		middleware.Logger(),
		NoCaching,
		sessionMiddleware.Middleware(models.Default),
	}

	wrapper := ServerInterfaceWrapper{Handler: server}
	e.GET(config.Server.BasePath+"/callback", wrapper.GetCallback, commonMiddleware...)
	e.POST(config.Server.BasePath+"/cli/login-complete", wrapper.PostCliLoginComplete, commonMiddleware...)
	e.POST(config.Server.BasePath+"/cli/login-init", wrapper.PostCliLoginInit, commonMiddleware...)
	e.GET(config.Server.BasePath+"/health", wrapper.GetHealth)
	e.GET(config.Server.BasePath+"/login", wrapper.GetLogin, commonMiddleware...)
	e.GET(config.Server.BasePath+"/logout", wrapper.GetLogout, middleware.Logger(), NoCaching)
	e.POST(config.Server.BasePath+"/logout-backend", wrapper.PostBackchannelLogout, middleware.Logger(), NoCaching)
	server.echo = e

	return server, nil
}
