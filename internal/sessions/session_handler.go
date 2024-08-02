package sessions

import (
	"log/slog"

	"github.com/labstack/echo/v4"
)

type SessionHandler struct {
}

func (s *SessionHandler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			slog.Info("SessionHandler: before")
			err := next(c)
			slog.Info("SessionHandler: after")
			return err
		}
	}
}

type SessionHandlerOption func(*SessionHandler) error

func NewSessionHandler(options ...SessionHandlerOption) SessionHandler {
	sh := SessionHandler{}
	for _, opt := range options {
		opt(&sh)
	}
	return sh
}
