// Package revproxy contains the definition of all routes, proxying and authentication
// performed by the reverse proxy that is part of the Renku gateway.
package revproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

type Revproxy struct {
	config   *config.RevproxyConfig
	sessions *sessions.SessionStore

	// Auth instances

	coreSvcIdTokenAuth             Auth
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
	// Intialize common reverse proxy middlewares
	fallbackProxy := proxyFromURL(r.config.RenkuBaseURL)
	renkuBaseProxyHost := setHost(r.config.RenkuBaseURL.Host)
	var gitlabProxy, gitlabProxyHost echo.MiddlewareFunc
	if r.config.ExternalGitlabURL != nil {
		gitlabProxy = proxyFromURL(r.config.ExternalGitlabURL)
		gitlabProxyHost = setHost(r.config.ExternalGitlabURL.Host)
	} else {
		gitlabProxy = fallbackProxy
		gitlabProxyHost = setHost(r.config.RenkuBaseURL.Host)
	}
	notebooksProxy := proxyFromURL(r.config.RenkuServices.Notebooks)
	kgProxy := proxyFromURL(r.config.RenkuServices.KG)
	webhookProxy := proxyFromURL(r.config.RenkuServices.Webhook)
	keycloakProxy := proxyFromURL(r.config.RenkuServices.Keycloak)
	keycloakProxyHost := setHost(r.config.RenkuServices.Keycloak.Host)
	dataServiceProxy := proxyFromURL(r.config.RenkuServices.DataService)
	uiServerProxy := proxyFromURL(r.config.RenkuServices.UIServer)
	searchProxy := proxyFromURL(r.config.RenkuServices.Search)

	// Initialize common authentication middleware
	coreSvcIdToken := r.coreSvcIdTokenAuth.Middleware()
	dataGitlabAccessToken := r.dataGitlabAccessTokenAuth.Middleware()
	gitlabToken := r.gitlabTokenAuth.Middleware()
	gitlabCliToken := r.gitlabCliTokenAuth.Middleware()
	notebooksRenkuAccessToken := r.notebooksRenkuAccessTokenAuth.Middleware()
	notebooksRenkuRefreshToken := r.notebooksRenkuRefreshTokenAuth.Middleware()
	notebooksRenkuIDToken := r.notebooksRenkuIDTokenAuth.Middleware()
	notebooksGitlabAccessToken := r.notebooksGitlabAccessTokenAuth.Middleware()
	renkuAccessToken := r.renkuAccessTokenAuth.Middleware()

	// Deny rules
	sk := e.Group("/api/data/user/secret_key", commonMiddlewares...)
	sk.GET("/", echo.NotFoundHandler)

	// Routing for Renku services
	e.Group("/api/notebooks", append(commonMiddlewares, notebooksRenkuAccessToken, notebooksRenkuRefreshToken, notebooksRenkuIDToken, notebooksGitlabAccessToken, notebooksAnonymousID(r.sessions), noCookies, stripPrefix("/api"), notebooksProxy)...)
	// /api/projects/:projectID/graph will is being deprecated in favour of /api/kg/webhooks, the old endpoint will remain for some time for backward compatibility
	e.Group("/api/projects/:projectID/graph", append(commonMiddlewares, gitlabToken, noCookies, kgProjectsGraphRewrites, webhookProxy)...)
	e.Group("/knowledge-graph", append(commonMiddlewares, gitlabToken, coreSvcIdToken, noCookies, kgProxy)...)
	e.Group("/api/kg/webhooks", append(commonMiddlewares, gitlabToken, noCookies, stripPrefix("/api/kg/webhooks"), webhookProxy)...)
	e.Group("/api/datasets", append(commonMiddlewares, noCookies, regexRewrite("^/api(.*)", "/knowledge-graph$1"), kgProxy)...)
	e.Group("/api/kg", append(commonMiddlewares, gitlabToken, noCookies, regexRewrite("^/api/kg(.*)", "/knowledge-graph$1"), kgProxy)...)
	e.Group("/api/data", append(commonMiddlewares, renkuAccessToken, dataGitlabAccessToken, notebooksRenkuRefreshToken, notebooksAnonymousID(r.sessions), dataServiceProxy)...)
	e.Group("/api/search", append(commonMiddlewares, renkuAccessToken, notebooksRenkuIDToken, notebooksAnonymousID(r.sessions), noCookies, searchProxy)...)
	// /api/kc is used only by the ui and no one else, will be removed when the gateway is in charge of user sessions
	e.Group("/api/kc", append(commonMiddlewares, stripPrefix("/api/kc"), renkuAccessToken, keycloakProxyHost, keycloakProxy)...)

	coreSvcProxyStartupCtx, cancel := context.WithTimeout(context.Background(), time.Duration(120)*time.Second)
	defer cancel()
	registerCoreSvcProxies(coreSvcProxyStartupCtx, e, r.config, append(commonMiddlewares, checkCoreServiceMetadataVersion(r.config.RenkuServices.Core.ServicePaths), coreSvcIdToken, gitlabToken, regexRewrite(`^/api/renku(?:/\d+)?((/|\?).*)??$`, "/renku$1"))...)

	// Routes that end up proxied to Gitlab
	if r.config.ExternalGitlabURL != nil {
		// Redirect "old" style bundled /gitlab pathing if an external Gitlab is used
		e.Group("/gitlab", append(commonMiddlewares, gitlabRedirect(r.config.ExternalGitlabURL.Host))...)
		e.Group("/api/graphql", append(commonMiddlewares, gitlabToken, gitlabProxyHost, gitlabProxy)...)
		e.Group("/api/direct", append(commonMiddlewares, stripPrefix("/api/direct"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/repos", append(commonMiddlewares, gitlabCliToken, noCookies, stripPrefix("/repos"), gitlabProxyHost, gitlabProxy)...)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", append(commonMiddlewares, gitlabToken, noCookies, regexRewrite("^/api(.*)", "/api/v4$1"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/ui-server/api/projects", append(commonMiddlewares, uiServerUpstreamExternalGitlabLocation(r.config.ExternalGitlabURL.Host), renkuAccessToken, dataGitlabAccessToken, uiServerProxy)...)
	} else {
		e.Group("/api/graphql", append(commonMiddlewares, gitlabToken, regexRewrite("^(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/api/direct", append(commonMiddlewares, regexRewrite("^/api/direct(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/repos", append(commonMiddlewares, gitlabCliToken, noCookies, regexRewrite("^/repos(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", append(commonMiddlewares, gitlabToken, noCookies, regexRewrite("^/api(.*)", "/gitlab/api/v4$1"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/ui-server/api/projects", append(commonMiddlewares, uiServerUpstreamInternalGitlabLocation(r.config.RenkuBaseURL.Host), renkuAccessToken, dataGitlabAccessToken, uiServerProxy)...)
	}

	// UI server webssockets
	e.Group("/ui-server/ws", append(commonMiddlewares, ensureSession(r.sessions), renkuAccessToken, uiServerProxy)...)
	// Some routes need to go to the UI server before they go to the specific Renku service
	e.Group("/ui-server/api/allows-iframe", append(commonMiddlewares, uiServerProxy)...)
	e.Group("/ui-server/api/last-searches/:length", append(commonMiddlewares, renkuAccessToken, uiServerProxy)...)
	e.Group("/ui-server/api/last-projects/:length", append(commonMiddlewares, renkuAccessToken, uiServerProxy)...)
	e.Group("/ui-server/api/renku/cache.files_upload", uiServerUpstreamCoreLocation(r.config.RenkuServices.Core.ServiceNames[0]), uiServerProxy)
	e.Group("/ui-server/api/kg/entities", append(commonMiddlewares, uiServerUpstreamKgLocation(r.config.RenkuServices.KG.Host), renkuAccessToken, dataGitlabAccessToken, uiServerProxy)...)

	// If nothing is matched from any of the routes above then fall back to the UI
	e.Group("/", append(commonMiddlewares, renkuBaseProxyHost, fallbackProxy)...)
}

func (r *Revproxy) initializeAuth() error {
	var err error

	r.coreSvcIdTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.IDTokenType), WithProviderID("renku"), WithTokenInjector(coreSvcRenkuIdTokenInjector))
	if err != nil {
		return err
	}
	r.dataGitlabAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), WithTokenInjector(dataServiceGitlabAccessTokenInjector))
	if err != nil {
		return err
	}
	r.gitlabTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), InjectBearerToken())
	if err != nil {
		return err
	}
	r.gitlabCliTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), WithTokenInjector(gitlabCliTokenInjector))
	if err != nil {
		return err
	}
	r.notebooksRenkuAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Access-Token"))
	if err != nil {
		return err
	}
	r.notebooksRenkuRefreshTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.RefreshTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Refresh-Token"))
	if err != nil {
		return err
	}
	r.notebooksRenkuIDTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.IDTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Id-Token"))
	if err != nil {
		return err
	}
	r.notebooksGitlabAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), WithTokenInjector(notebooksGitlabAccessTokenInjector))
	if err != nil {
		return err
	}
	r.renkuAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("renku"), InjectBearerToken())
	if err != nil {
		return err
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
