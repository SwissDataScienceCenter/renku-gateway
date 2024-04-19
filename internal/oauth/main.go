// Package oauth contains routes and authentication performed by
// the reverse proxy as an OAuth2 client for configured
// third party services.
package oauth

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/labstack/echo/v4"
)

type OAuthServer struct {
	config *config.OAuthClientsConfig
}

func (r *OAuthServer) RegisterHandlers(server *echo.Echo, commonMiddlewares ...echo.MiddlewareFunc) {
	e := server.Group("/api/oauth")
	e.Use(commonMiddlewares...)

	e.GET("/hello", func(c echo.Context) error {
		return c.String(200, "Hello")
	})
}

func NewServer(config *config.OAuthClientsConfig) OAuthServer {
	return OAuthServer{config}
}
