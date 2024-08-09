package utils

import "github.com/labstack/echo/v4"

func GetRequestID(c echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}
