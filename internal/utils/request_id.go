package utils

import "github.com/labstack/echo/v5"

func GetRequestID(c *echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}
