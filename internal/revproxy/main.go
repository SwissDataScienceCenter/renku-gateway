// Package revproxy contains the definition of all routes, proxying and authentication
// performed by the reverse proxy that is part of the Renku gateway.
package revproxy

import (
	"fmt"
	"path"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/redirects"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

type Revproxy struct {
	config    *config.RevproxyConfig
	sessions  *sessions.SessionStore
	redirects *redirects.RedirectStore

	// Auth instances

	dataGitlabAccessTokenAuth      Auth
	gitlabTokenAuth                Auth
	gitlabCliTokenAuth             Auth
	notebooksRenkuAccessTokenAuth  Auth
	notebooksRenkuRefreshTokenAuth Auth
	notebooksRenkuIDTokenAuth      Auth
	notebooksGitlabAccessTokenAuth Auth
	renkuAccessTokenAuth           Auth
}

func (r *Revproxy) RegisterHandlers(e *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	// Initialize common reverse proxy middlewares
	fallbackProxy := proxyFromURL(r.config.RenkuBaseURL)
	renkuBaseProxyHost := setHost(r.config.RenkuBaseURL.Host)
	keycloakProxy := proxyFromURL(r.config.RenkuServices.Keycloak)
	keycloakProxyHost := setHost(r.config.RenkuServices.Keycloak.Host)
	dataServiceProxy := proxyFromURL(r.config.RenkuServices.DataService)
	uiServerProxy := proxyFromURL(r.config.RenkuServices.UIServer)

	// Deny rules
	sk := e.Group("/api/data/user/secret_key", commonMiddlewares...)
	sk.GET("/", echo.NotFoundHandler)

	// Redirects store middleware
	if r.redirects != nil {
		redirectMiddleware := r.redirects.Middleware()
		redirectPath := path.Join(r.redirects.PathPrefix, ":projectPath")
		e.Group(redirectPath, append(commonMiddlewares, renkuBaseProxyHost, redirectMiddleware, fallbackProxy)...)
	}

	if r.config.EnableInternalGitlab {
		// This whole branch of else-if should be removed when the Gitlab is retired.
		// Initialize common authentication middleware
		notebooksRenkuRefreshToken := r.notebooksRenkuRefreshTokenAuth.Middleware()
		renkuAccessToken := r.renkuAccessTokenAuth.Middleware()
		dataGitlabAccessToken := r.dataGitlabAccessTokenAuth.Middleware()

		// Routing for Renku services
		// Notebooks is being routed to data service now
		e.Group("/api/notebooks", append(commonMiddlewares, renkuAccessToken, dataGitlabAccessToken, notebooksRenkuRefreshToken, notebooksAnonymousID(r.sessions), regexRewrite("^/api/notebooks(.*)", "/api/data/notebooks$1"), dataServiceProxy)...)
		e.Group("/api/data", append(commonMiddlewares, renkuAccessToken, dataGitlabAccessToken, notebooksRenkuRefreshToken, notebooksAnonymousID(r.sessions), dataServiceProxy)...)
		// /api/kc is used only by the ui and no one else, will be removed when the gateway is in charge of user sessions
		e.Group("/api/kc", append(commonMiddlewares, stripPrefix("/api/kc"), renkuAccessToken, keycloakProxyHost, keycloakProxy)...)

		// UI server websockets
		e.Group("/ui-server/ws", append(commonMiddlewares, ensureSession(r.sessions), renkuAccessToken, uiServerProxy)...)
		// Some routes need to go to the UI server before they go to the specific Renku service
		e.Group("/ui-server/api/allows-iframe", append(commonMiddlewares, uiServerProxy)...)
	} else {
		// Both the v1 services and internal gitlab are disabled
		// Initialize common authentication middleware
		notebooksRenkuRefreshToken := r.notebooksRenkuRefreshTokenAuth.Middleware()
		renkuAccessToken := r.renkuAccessTokenAuth.Middleware()

		// Routing for Renku services
		// Notebooks is being routed to data service now
		e.Group("/api/notebooks", append(commonMiddlewares, renkuAccessToken, notebooksRenkuRefreshToken, notebooksAnonymousID(r.sessions), regexRewrite("^/api/notebooks(.*)", "/api/data/notebooks$1"), dataServiceProxy)...)
		e.Group("/api/data", append(commonMiddlewares, renkuAccessToken, notebooksRenkuRefreshToken, notebooksAnonymousID(r.sessions), dataServiceProxy)...)
		// /api/kc is used only by the ui and no one else, will be removed when the gateway is in charge of user sessions
		e.Group("/api/kc", append(commonMiddlewares, stripPrefix("/api/kc"), renkuAccessToken, keycloakProxyHost, keycloakProxy)...)

		// UI server websockets
		e.Group("/ui-server/ws", append(commonMiddlewares, ensureSession(r.sessions), renkuAccessToken, uiServerProxy)...)
		// Some routes need to go to the UI server before they go to the specific Renku service
		e.Group("/ui-server/api/allows-iframe", append(commonMiddlewares, uiServerProxy)...)
	}

	// If nothing is matched from any of the routes above then fall back to the UI
	e.Group("/", append(commonMiddlewares, renkuBaseProxyHost, fallbackProxy)...)
}

func (r *Revproxy) initializeAuth() error {
	var err error

	// Initialize auth for v2 services first
	r.renkuAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("renku"), InjectBearerToken())
	if err != nil {
		return err
	}
	r.notebooksRenkuRefreshTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.RefreshTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Refresh-Token"))
	if err != nil {
		return err
	}

	if r.config.EnableInternalGitlab {
		r.dataGitlabAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), WithTokenInjector(dataServiceGitlabAccessTokenInjector))
		if err != nil {
			return err
		}
	}

	return nil
}

type RevproxyOption func(*Revproxy)

func WithConfig(revproxyConfig config.RevproxyConfig) RevproxyOption {
	return func(l *Revproxy) {
		l.config = &revproxyConfig
	}
}

func WithSessionStore(sessions *sessions.SessionStore) RevproxyOption {
	return func(l *Revproxy) {
		l.sessions = sessions
	}
}

func WithRedirectsStore(redirects *redirects.RedirectStore) RevproxyOption {
	return func(l *Revproxy) {
		l.redirects = redirects
	}
}

func NewServer(options ...RevproxyOption) (*Revproxy, error) {
	server := Revproxy{}
	for _, opt := range options {
		opt(&server)
	}
	if server.config == nil {
		return &Revproxy{}, fmt.Errorf("revproxy config not provided")
	}
	if server.sessions == nil {
		return &Revproxy{}, fmt.Errorf("session handler not initialized")
	}
	err := server.initializeAuth()
	if err != nil {
		return &Revproxy{}, err
	}
	return &server, nil
}
