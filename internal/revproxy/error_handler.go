package revproxy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// The same error content as the data services API
const gwBaseErrorCode int = 6000

type errorContent struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
	TraceId string `json:"trace_id,omitempty"`
}

type errorResponse struct {
	Error errorContent `json:"error"`
}

// Adapted from https://echo.labstack.com/docs/error-handling
func ErrorHandler(err error, c echo.Context) {
	if c.Response() != nil && c.Response().Committed {
		return // response has been already sent to the client by handler or some middleware
	}

	accept := c.Request().Header.Get("Accept")
	isHTML := strings.Contains(accept, echo.MIMETextHTML)

	// If the accept header is html then we fall back to the default handler (for now).
	// If the acceptt header is not html or is blank we return json
	if isHTML {
		c.Echo().DefaultHTTPErrorHandler(err, c)
		return
	}

	code := http.StatusInternalServerError
	message := err.Error()
	var he *echo.HTTPError
	if errors.As(err, &he) { // find error in an error chain that implements HTTPError
		if tmp := he.Code; tmp != 0 {
			code = tmp
		}
		if msg := fmt.Sprintf("%v", he.Message); len(msg) > 0 {
			message = msg
		}
	}

	var cErr error
	if c.Request().Method == http.MethodHead {
		cErr = c.NoContent(code)
	} else {
		cErr = c.JSON(code, errorResponse{Error: errorContent{Code: gwBaseErrorCode + code, Message: message}})
	}
	if cErr != nil {
		c.Logger().Error("failed to send error page to client", "error", errors.Join(err, cErr))
	}
}
