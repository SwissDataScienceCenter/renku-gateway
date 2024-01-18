// Package main contains the definition of all routes, proxying and authentication
// performed by the reverse proxy that is part of the Renku gateway.
package revproxy

import (
	"context"
	"net/url"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type tokenStore interface {
	models.AccessTokenGetter
	models.RefreshTokenGetter
}

type Revproxy struct {
	config *config.RevproxyConfig
	// to do make it so that tokens are injected from redis directly
	db tokenStore	
}

func (r *Revproxy) RegisterHandlers(e *echo.Echo) {
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
	logger := middleware.Logger()

	// Initialize common authentication middleware
	notebooksAuth := printMsg("auth")
	dataAuth := printMsg("auth")
	renkuAuth := printMsg("auth")
	gitlabAuth := printMsg("auth")
	cliGitlabAuth := printMsg("auth")

	// Routing for Renku services
	e.Group("/api/notebooks", logger, notebooksAuth, noCookies, stripPrefix("/api"), notebooksProxy)
	// /api/projects/:projectID/graph will is being deprecated in favour of /api/kg/webhooks, the old endpoint will remain for some time for backward compatibility
	e.Group("/api/projects/:projectID/graph", logger, gitlabAuth, noCookies, kgProjectsGraphRewrites, webhookProxy)
	e.Group("/api/kg/webhooks", logger, gitlabAuth, noCookies, stripPrefix("/api/kg/webhooks"), webhookProxy)
	e.Group("/api/datasets", logger, noCookies, regexRewrite("^/api(.*)", "/knowledge-graph$1"), kgProxy)
	e.Group("/api/kg", logger, gitlabAuth, noCookies, regexRewrite("^/api/kg(.*)", "/knowledge-graph$1"), kgProxy)
	e.Group("/api/data", logger, dataAuth, noCookies, dataServiceProxy)
	// /api/kc is used only by the ui and no one else, will be removed when the gateway is in charge of user sessions
	e.Group("/api/kc", logger, stripPrefix("/api/kc"), keycloakProxyHost, keycloakProxy)

	coreSvcProxyStartupCtx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	registerCoreSvcProxies(coreSvcProxyStartupCtx, e, r.config, logger, checkCoreServiceMetadataVersion(r.config.RenkuServices.Core.ServicePaths), renkuAuth, noCookies, regexRewrite(`^/api/renku(?:/\d+)?((/|\?).*)??$`, "/renku$1"))

	// Routes that end up proxied to Gitlab
	if r.config.ExternalGitlabURL != nil {
		// Redirect "old" style bundled /gitlab pathing if an external Gitlab is used
		e.Group("/gitlab", logger, gitlabRedirect(r.config.ExternalGitlabURL.Host))
		e.Group("/api/graphql", logger, gitlabAuth, gitlabProxyHost, gitlabProxy)
		e.Group("/api/direct", logger, stripPrefix("/api/direct"), gitlabProxyHost, gitlabProxy)
		e.Group("/repos", logger, cliGitlabAuth, noCookies, stripPrefix("/repos"), gitlabProxyHost, gitlabProxy)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", logger, gitlabAuth, noCookies, regexRewrite("^/api(.*)", "/api/v4$1"), gitlabProxyHost, gitlabProxy)
	} else {
		e.Group("/api/graphql", logger, gitlabAuth, regexRewrite("^(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)
		e.Group("/api/direct", logger, regexRewrite("^/api/direct(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)
		e.Group("/repos", logger, cliGitlabAuth, noCookies, regexRewrite("^/repos(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", logger, gitlabAuth, noCookies, regexRewrite("^/api(.*)", "/gitlab/api/v4$1"), gitlabProxyHost, gitlabProxy)
	}

	// If nothing is matched from any of the routes above then fall back to the UI
	e.Group("/", logger, renkuBaseProxyHost, fallbackProxy)
}

func NewServer(revproxyConfig *config.RevproxyConfig) Revproxy {
	return Revproxy{config: revproxyConfig}
}

// addQueryParams makes a copy of the provided URL, adds the query parameters
// and returns a url with the added parameters. The original URL is left unchanged.
func addQueryParams(url *url.URL, params map[string]string) *url.URL {
	newURL := *url
	query := newURL.Query()
	for k, v := range params {
		query.Add(k, v)
	}
	newURL.RawQuery = query.Encode()
	return &newURL
}
