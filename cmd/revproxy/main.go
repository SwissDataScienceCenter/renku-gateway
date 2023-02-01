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
	coreProxy := proxyFromURL(config.RenkuServices.Core)
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
	e.Use(middleware.Recover())

	// Routing for Renku services
	e.Group("/api/auth", logger, authSvcProxy)
	e.Group("/api/notebooks", logger, notebooksAuth, noCookies, stripPrefix("/api"), notebooksProxy)
	e.Group("/api/projects/:projectID/graph", logger, gitlabAuth, noCookies, kgProjectsGraphRewrites, webhookProxy)
	e.Group("/api/datasets", logger, noCookies, regexRewrite("^/api/(.*)", "/knowledge-graph/$1"), kgProxy)
	e.Group("/api/kg", logger, gitlabAuth, noCookies, regexRewrite("^/api/kg/(.*)", "/knowledge-graph/$1"), kgProxy)
	e.Group("/api/renku", logger, renkuAuth, noCookies, stripPrefix("/api"), coreProxy)

	// Routes that end up proxied to Gitlab
	if config.ExternalGitlabURL != nil {
		// Redirect "old" style bundled /gitlab pathing if an external Gitlab is used
		e.Group("/gitlab", logger, stripPrefix("/gitlab"), gitlabProxyHost, gitlabProxy)
		e.Group("/api/graphql", logger, gitlabAuth, gitlabProxyHost, gitlabProxy)
		e.Group("/api/direct", logger, stripPrefix("/api/direct"), gitlabProxyHost, gitlabProxy)
		e.Group("/api/repos", logger, cliGitlabAuth, noCookies, stripPrefix("/api/repos"), gitlabProxyHost, gitlabProxy)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", logger, gitlabAuth, noCookies, regexRewrite("^/api/(.*)", "/api/v4/$1"), gitlabProxyHost, gitlabProxy)
	} else {
		e.Group("/api/graphql", logger, gitlabAuth, regexRewrite("^/(.*)", "/gitlab/$1"), gitlabProxyHost, gitlabProxy)
		e.Group("/api/direct", logger, regexRewrite("^/api/direct/(.*)", "/gitlab/$1"), gitlabProxyHost, gitlabProxy)
		e.Group("/api/repos", logger, cliGitlabAuth, noCookies, regexRewrite("^/api/repos/(.*)", "/gitlab/$1"), gitlabProxyHost, gitlabProxy)
		// If nothing is matched in any other more specific /api route then fall back to Gitlab
		e.Group("/api", logger, gitlabAuth, noCookies, regexRewrite("^/api/(.*)", "/gitlab/api/v4/$1"), gitlabProxyHost, gitlabProxy)
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
	e.Logger.Printf("Starting server with config: %+v", config)
	go func() {
		if err := e.Start(fmt.Sprintf(":%d", config.Port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal(err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
