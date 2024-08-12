// Package revproxy contains the definition of all routes, proxying and authentication
// performed by the reverse proxy that is part of the Renku gateway.
package revproxy

import (
	"fmt"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

type Revproxy struct {
	config   *config.RevproxyConfig
	sessions *sessions.SessionStore

	// Auth instances

	dataRenkuAccessTokenAuth  Auth
	dataGitlabAccessTokenAuth Auth
}

func (r *Revproxy) RegisterHandlers(e *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	// Intialize common reverse proxy middlewares
	// fallbackProxy := proxyFromURL(r.config.RenkuBaseURL)
	// renkuBaseProxyHost := setHost(r.config.RenkuBaseURL.Host)
	// var gitlabProxy, gitlabProxyHost echo.MiddlewareFunc
	// if r.config.ExternalGitlabURL != nil {
	// 	gitlabProxy = proxyFromURL(r.config.ExternalGitlabURL)
	// 	gitlabProxyHost = setHost(r.config.ExternalGitlabURL.Host)
	// } else {
	// 	gitlabProxy = fallbackProxy
	// 	gitlabProxyHost = setHost(r.config.RenkuBaseURL.Host)
	// }
	// notebooksProxy := proxyFromURL(r.config.RenkuServices.Notebooks)
	// kgProxy := proxyFromURL(r.config.RenkuServices.KG)
	// webhookProxy := proxyFromURL(r.config.RenkuServices.Webhook)
	// keycloakProxy := proxyFromURL(r.config.RenkuServices.Keycloak)
	// keycloakProxyHost := setHost(r.config.RenkuServices.Keycloak.Host)
	dataServiceProxy := proxyFromURL(r.config.RenkuServices.DataService)
	// uiServerProxy := proxyFromURL(r.config.RenkuServices.UIServer)

	// Initialize common authentication middleware
	// notebooksAuthAccessToken := NewAuth(WithTokenType(models.AccessTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Access-Token")).Middleware()
	// notebooksAuthIDToken := NewAuth(WithTokenType(models.IDTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Id-Token")).Middleware()
	// notebooksAuthRefreshToken := NewAuth(WithTokenType(models.RefreshTokenType), WithProviderID("renku"), InjectInHeader("Renku-Auth-Refresh-Token")).Middleware()
	// notebooksGitlabAccessToken := NewAuth(WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), WithTokenHandler(notebooksGitlabAccessTokenHandler)).Middleware()
	dataRenkuAccessToken := r.dataRenkuAccessTokenAuth.Middleware()
	dataGitlabAccessToken := r.dataGitlabAccessTokenAuth.Middleware()
	// coreSvcIdToken := NewAuth(WithTokenType(models.IDTokenType), WithProviderID("renku"), InjectInHeader("Renku-User")).Middleware()
	// gitlabAuth := NewAuth(WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), InjectBearerToken()).Middleware()
	// gitlabCliAuth := NewAuth(WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), WithTokenHandler(gitlabCliTokenHandler)).Middleware()

	e.Group("/api/data", append(commonMiddlewares, dataRenkuAccessToken, dataGitlabAccessToken, noCookies, dataServiceProxy)...)

	// // Routing for Renku services
	// e.Group("/api/notebooks", append(commonMiddlewares, notebooksAuthAccessToken, notebooksAuthIDToken, notebooksAuthRefreshToken, notebooksGitlabAccessToken, notebooksAnonymousID, noCookies, stripPrefix("/api"), notebooksProxy)...)
	// // /api/projects/:projectID/graph will is being deprecated in favour of /api/kg/webhooks, the old endpoint will remain for some time for backward compatibility
	// e.Group("/api/projects/:projectID/graph", append(commonMiddlewares, gitlabAuth, noCookies, kgProjectsGraphRewrites, webhookProxy)...)
	// e.Group("/knowledge-graph", append(commonMiddlewares, gitlabAuth, coreSvcIdToken, noCookies, kgProxy)...)
	// e.Group("/api/kg/webhooks", append(commonMiddlewares, gitlabAuth, noCookies, stripPrefix("/api/kg/webhooks"), webhookProxy)...)
	// e.Group("/api/datasets", append(commonMiddlewares, noCookies, regexRewrite("^/api(.*)", "/knowledge-graph$1"), kgProxy)...)
	// e.Group("/api/kg", append(commonMiddlewares, gitlabAuth, noCookies, regexRewrite("^/api/kg(.*)", "/knowledge-graph$1"), kgProxy)...)
	// e.Group("/api/data", append(commonMiddlewares, dataRenkuAccessToken, dataGitlabAccessToken, noCookies, dataServiceProxy)...)
	// // /api/kc is used only by the ui and no one else, will be removed when the gateway is in charge of user sessions
	// e.Group("/api/kc", append(commonMiddlewares, stripPrefix("/api/kc"), keycloakProxyHost, keycloakProxy)...)

	// coreSvcProxyStartupCtx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	// defer cancel()
	// registerCoreSvcProxies(coreSvcProxyStartupCtx, e, r.config, append(commonMiddlewares, checkCoreServiceMetadataVersion(r.config.RenkuServices.Core.ServicePaths), coreSvcIdToken, gitlabAuth, regexRewrite(`^/api/renku(?:/\d+)?((/|\?).*)??$`, "/renku$1"))...)

	// // Routes that end up proxied to Gitlab
	// if r.config.ExternalGitlabURL != nil {
	// 	// Redirect "old" style bundled /gitlab pathing if an external Gitlab is used
	// 	e.Group("/gitlab", append(commonMiddlewares, gitlabRedirect(r.config.ExternalGitlabURL.Host))...)
	// 	e.Group("/api/graphql", append(commonMiddlewares, gitlabAuth, gitlabProxyHost, gitlabProxy)...)
	// 	e.Group("/api/direct", append(commonMiddlewares, stripPrefix("/api/direct"), gitlabProxyHost, gitlabProxy)...)
	// 	e.Group("/repos", append(commonMiddlewares, gitlabCliAuth, noCookies, stripPrefix("/repos"), gitlabProxyHost, gitlabProxy)...)
	// 	// If nothing is matched in any other more specific /api route then fall back to Gitlab
	// 	e.Group("/api", append(commonMiddlewares, gitlabAuth, noCookies, regexRewrite("^/api(.*)", "/api/v4$1"), gitlabProxyHost, gitlabProxy)...)
	// 	e.Group("/ui-server/api/projects/:projectName", append(commonMiddlewares, uiServerUpstreamExternalGitlabLocation(r.config.ExternalGitlabURL.Host), uiServerProxy)...)
	// } else {
	// 	e.Group("/api/graphql", append(commonMiddlewares, gitlabAuth, regexRewrite("^(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
	// 	e.Group("/api/direct", append(commonMiddlewares, regexRewrite("^/api/direct(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
	// 	e.Group("/repos", append(commonMiddlewares, gitlabCliAuth, noCookies, regexRewrite("^/repos(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
	// 	// If nothing is matched in any other more specific /api route then fall back to Gitlab
	// 	e.Group("/api", append(commonMiddlewares, gitlabAuth, noCookies, regexRewrite("^/api(.*)", "/gitlab/api/v4$1"), gitlabProxyHost, gitlabProxy)...)
	// 	e.Group("/ui-server/api/projects/:projectName", append(commonMiddlewares, uiServerUpstreamInternalGitlabLocation(r.config.RenkuBaseURL.Host), uiServerProxy)...)
	// }

	// // UI server webssockets
	// e.Group("/ui-server/ws", append(commonMiddlewares, uiServerProxy)...)
	// // Some routes need to go to the UI server before they go to the specific Renku service
	// e.Group("/ui-server/api/last-searches/:length", append(commonMiddlewares, uiServerProxy)...)
	// e.Group("/ui-server/api/last-projects/:length", append(commonMiddlewares, uiServerProxy)...)
	// // e.Group("/ui-server/api/renku/cache.files_upload", uiServerUpstreamCoreLocation(r.config.RenkuServices.Core.ServicePaths[0].Host), uiServerProxy)
	// e.Group("/ui-server/api/kg/entities", append(commonMiddlewares, uiServerUpstreamKgLocation(r.config.RenkuServices.KG.Host), uiServerProxy)...)

	// // If nothing is matched from any of the routes above then fall back to the UI
	// e.Group("/", append(commonMiddlewares, renkuBaseProxyHost, fallbackProxy)...)
}

func (r *Revproxy) initializeAuth() error {
	var err error

	r.dataRenkuAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("renku"), InjectBearerToken())
	if err != nil {
		return err
	}
	r.dataGitlabAccessTokenAuth, err = NewAuth(AuthWithSessionStore(r.sessions), WithTokenType(models.AccessTokenType), WithProviderID("gitlab"), InjectInHeader("Gitlab-Access-Token"))
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
