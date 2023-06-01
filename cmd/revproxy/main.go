// Package main contains the definition of all routes, proxying and authentication
// performed by the reverse proxy that is part of the Renku gateway.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func setupServer(config revProxyConfig) *echo.Echo {
	// Intialize common reverse proxy middlewares
	fallbackProxy := proxyFromURL(config.RenkuBaseURL)
	renkuBaseProxyHost := setHost(config.RenkuBaseURL.Host)
	var gitlabProxy, gitlabProxyHost echo.MiddlewareFunc
	if config.ExternalGitlabURL != nil {
		gitlabProxy = proxyFromURL(config.ExternalGitlabURL)
		gitlabProxyHost = setHost(config.ExternalGitlabURL.Host)
	} else {
		gitlabProxy = fallbackProxy
		gitlabProxyHost = setHost(config.RenkuBaseURL.Host)
	}
	notebooksProxy := proxyFromURL(config.RenkuServices.Notebooks)
	authSvcProxy := proxyFromURL(config.RenkuServices.Auth)
	kgProxy := proxyFromURL(config.RenkuServices.KG)
	webhookProxy := proxyFromURL(config.RenkuServices.Webhook)
	logger := middleware.Logger()

	// Initialize common authentication middleware
	notebooksAuth := authenticate(AddQueryParams(config.RenkuServices.Auth, map[string]string{"auth": "notebook"}), "Renku-Auth-Access-Token", "Renku-Auth-Id-Token", "Renku-Auth-Git-Credentials", "Renku-Auth-Anon-Id", "Renku-Auth-Refresh-Token")
	renkuAuth := authenticate(AddQueryParams(config.RenkuServices.Auth, map[string]string{"auth": "renku"}), "Authorization", "Renku-user-id", "Renku-user-fullname", "Renku-user-email")
	gitlabAuth := authenticate(AddQueryParams(config.RenkuServices.Auth, map[string]string{"auth": "gitlab"}), "Authorization")
	cliGitlabAuth := authenticate(AddQueryParams(config.RenkuServices.Auth, map[string]string{"auth": "cli-gitlab"}), "Authorization")

	// Server instance
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	if config.RateLimits.Enabled {
		e.Use(middleware.RateLimiter(
			middleware.NewRateLimiterMemoryStoreWithConfig(
				middleware.RateLimiterMemoryStoreConfig{
					Rate:      rate.Limit(config.RateLimits.Rate),
					Burst:     config.RateLimits.Burst,
					ExpiresIn: 3 * time.Minute,
				}),
		),
		)
	}

	// Routing for Renku services
	e.Group("/api/auth", logger, authSvcProxy)
	e.Group("/api/notebooks", logger, notebooksAuth, noCookies, stripPrefix("/api"), notebooksProxy)
	// /api/projects/:projectID/graph will is being deprecated in favour of /api/kg/webhooks, the old endpoint will remain for some time for backward compatibility
	e.Group("/api/projects/:projectID/graph", logger, gitlabAuth, noCookies, kgProjectsGraphRewrites, webhookProxy)
	e.Group("/api/kg/webhooks", logger, gitlabAuth, noCookies, stripPrefix("/api/kg/webhooks"), webhookProxy)
	e.Group("/api/datasets", logger, noCookies, regexRewrite("^/api(.*)", "/knowledge-graph$1"), kgProxy)
	e.Group("/api/kg", logger, gitlabAuth, noCookies, regexRewrite("^/api/kg(.*)", "/knowledge-graph$1"), kgProxy)

	registerCoreSvcProxies(e, config, logger, renkuAuth, noCookies, regexRewrite(`^/api/renku(?:/\d+)?((/|\?).*)??$`, "/renku$1"))

	// Routes that end up proxied to Gitlab
	if config.ExternalGitlabURL != nil {
		// Redirect "old" style bundled /gitlab pathing if an external Gitlab is used
		e.Group("/gitlab", logger, stripPrefix("/gitlab"), gitlabProxyHost, gitlabProxy)
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

	// Reverse proxy specific endpoints
	rp := e.Group("/revproxy")
	rp.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return e
}

func main() {
	config := getConfig()
	e := setupServer(config)
	// Start API server
	e.Logger.Printf("Starting server with config: %+v", config)
	go func() {
		if err := e.Start(fmt.Sprintf(":%d", config.Port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal(err)
		}
	}()
	// Start metrics server if enabled
	var metricsServer *echo.Echo
	if config.Metrics.Enabled {
		metricsServer = getMetricsServer(e, config.Metrics.Port)
		go func() {
			if err := metricsServer.Start(fmt.Sprintf(":%d", config.Metrics.Port)); err != nil && err != http.ErrServerClosed {
				metricsServer.Logger.Fatal(err)
			}
		}()
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit // Wait for interrupt signal from OS
	// Start shutting down servers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
	if config.Metrics.Enabled {
		if err := metricsServer.Shutdown(ctx); err != nil {
			metricsServer.Logger.Fatal(err)
		}
	}
}
