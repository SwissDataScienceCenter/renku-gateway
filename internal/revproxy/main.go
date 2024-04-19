// Package revproxy contains the definition of all routes, proxying and authentication
// performed by the reverse proxy that is part of the Renku gateway.
package revproxy

import (
	"context"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/labstack/echo/v4"
)

type Revproxy struct {
	config *config.RevproxyConfig
}

func (r *Revproxy) RegisterHandlers(e *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	// Intialize common reverse proxy middlewares
	fallbackProxy := proxyFromURL(r.config.RenkuBaseURL)
	gatewayAuthProxy := proxyFromURL(r.config.RenkuServices.GatewayAuth)
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
	searchProxy := proxyFromURL(r.config.RenkuServices.Search)

	// Initialize legacy authentication middleware
	baseAuth := LegacyAuth(r.config.RenkuServices.GatewayAuth)
	notebooksAuth := baseAuth("notebook", "Renku-Auth-Access-Token", "Renku-Auth-Id-Token", "Renku-Auth-Git-Credentials", "Renku-Auth-Anon-Id", "Renku-Auth-Refresh-Token")
	searchAuth := baseAuth("search", "Renku-Auth-Id-Token", "Renku-Auth-Anon-Id")
	dataAuth := baseAuth("keycloak_gitlab", "Authorization", "Gitlab-Access-Token")
	renkuAuth := baseAuth("renku", "Authorization", "Renku-user-id", "Renku-user-fullname", "Renku-user-email")
	gitlabAuth := baseAuth("gitlab", "Authorization")
	cliGitlabAuth := baseAuth("cli-gitlab", "Authorization")

	// Routing for Renku services
	e.Group("/api/auth", append(commonMiddlewares, gatewayAuthProxy)...)
	e.Group("/api/notebooks", append(commonMiddlewares, notebooksAuth, stripPrefix("/api"), notebooksProxy)...)
	// /api/projects/:projectID/graph will is being deprecated in favour of /api/kg/webhooks, the old endpoint will remain for some time for backward compatibility
	e.Group("/api/projects/:projectID/graph", append(commonMiddlewares, gitlabAuth, noCookies, kgProjectsGraphRewrites, webhookProxy)...)
	e.Group("/api/kg/webhooks", append(commonMiddlewares, gitlabAuth, noCookies, stripPrefix("/api/kg/webhooks"), webhookProxy)...)
	e.Group("/api/datasets", append(commonMiddlewares, noCookies, regexRewrite("^/api(.*)", "/knowledge-graph$1"), kgProxy)...)
	e.Group("/api/kg", append(commonMiddlewares, gitlabAuth, noCookies, regexRewrite("^/api/kg(.*)", "/knowledge-graph$1"), kgProxy)...)
	e.Group("/api/data", append(commonMiddlewares, dataAuth, noCookies, dataServiceProxy)...)
	e.Group("/api/search", append(commonMiddlewares, searchAuth, noCookies, stripPrefix("/api"), searchProxy)...)
	// /api/kc is used only by the ui and no one else, will be removed when the gateway is in charge of user sessions
	e.Group("/api/kc", append(commonMiddlewares, stripPrefix("/api/kc"), keycloakProxyHost, keycloakProxy)...)

	coreSvcProxyStartupCtx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	registerCoreSvcProxies(coreSvcProxyStartupCtx, e, r.config, append(commonMiddlewares, checkCoreServiceMetadataVersion(r.config.RenkuServices.Core.ServicePaths), renkuAuth, noCookies, regexRewrite(`^/api/renku(?:/\d+)?((/|\?).*)??$`, "/renku$1"))...)

	// Routes that end up proxied to Gitlab
	if r.config.ExternalGitlabURL != nil {
		// Redirect "old" style bundled /gitlab pathing if an external Gitlab is used
		e.Group("/gitlab", append(commonMiddlewares, gitlabRedirect(r.config.ExternalGitlabURL.Host))...)
		e.Group("/api/graphql", append(commonMiddlewares, gitlabAuth, gitlabProxyHost, gitlabProxy)...)
		e.Group("/api/direct", append(commonMiddlewares, stripPrefix("/api/direct"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/repos", append(commonMiddlewares, cliGitlabAuth, noCookies, stripPrefix("/repos"), gitlabProxyHost, gitlabProxy)...)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", append(commonMiddlewares, gitlabAuth, noCookies, regexRewrite("^/api(.*)", "/api/v4$1"), gitlabProxyHost, gitlabProxy)...)
	} else {
		e.Group("/api/graphql", append(commonMiddlewares, gitlabAuth, regexRewrite("^(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/api/direct", append(commonMiddlewares, regexRewrite("^/api/direct(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
		e.Group("/repos", append(commonMiddlewares, cliGitlabAuth, noCookies, regexRewrite("^/repos(.*)", "/gitlab$1"), gitlabProxyHost, gitlabProxy)...)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", append(commonMiddlewares, gitlabAuth, noCookies, regexRewrite("^/api(.*)", "/gitlab/api/v4$1"), gitlabProxyHost, gitlabProxy)...)
	}

	// If nothing is matched from any of the routes above then fall back to the UI
	e.Group("/", append(commonMiddlewares, renkuBaseProxyHost, fallbackProxy)...)
}

func NewServer(revproxyConfig *config.RevproxyConfig) Revproxy {
	return Revproxy{config: revproxyConfig}
}
