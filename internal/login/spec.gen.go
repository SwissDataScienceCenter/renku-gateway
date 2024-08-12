// Package login provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version (devel) DO NOT EDIT.
package login

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime"
)

// GetCallbackParams defines parameters for GetCallback.
type GetCallbackParams struct {
	Code         string  `form:"code" json:"code"`
	State        string  `form:"state" json:"state"`
	SessionState *string `form:"session_state,omitempty" json:"session_state,omitempty"`
}

// GetLoginParams defines parameters for GetLogin.
type GetLoginParams struct {
	RedirectUrl *string   `form:"redirect_url,omitempty" json:"redirect_url,omitempty"`
	ProviderId  *[]string `form:"provider_id,omitempty" json:"provider_id,omitempty"`
}

// GetLogoutParams defines parameters for GetLogout.
type GetLogoutParams struct {
	RedirectUrl *string `form:"redirect_url,omitempty" json:"redirect_url,omitempty"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /callback)
	GetCallback(ctx echo.Context, params GetCallbackParams) error

	// (GET /health)
	GetHealth(ctx echo.Context) error

	// (GET /login)
	GetLogin(ctx echo.Context, params GetLoginParams) error

	// (GET /logout)
	GetLogout(ctx echo.Context, params GetLogoutParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetCallback converts echo context to params.
func (w *ServerInterfaceWrapper) GetCallback(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetCallbackParams
	// ------------- Required query parameter "code" -------------

	err = runtime.BindQueryParameter("form", true, true, "code", ctx.QueryParams(), &params.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter code: %s", err))
	}

	// ------------- Required query parameter "state" -------------

	err = runtime.BindQueryParameter("form", true, true, "state", ctx.QueryParams(), &params.State)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter state: %s", err))
	}

	// ------------- Optional query parameter "session_state" -------------

	err = runtime.BindQueryParameter("form", true, false, "session_state", ctx.QueryParams(), &params.SessionState)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter session_state: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetCallback(ctx, params)
	return err
}

// GetHealth converts echo context to params.
func (w *ServerInterfaceWrapper) GetHealth(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetHealth(ctx)
	return err
}

// GetLogin converts echo context to params.
func (w *ServerInterfaceWrapper) GetLogin(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetLoginParams
	// ------------- Optional query parameter "redirect_url" -------------

	err = runtime.BindQueryParameter("form", true, false, "redirect_url", ctx.QueryParams(), &params.RedirectUrl)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter redirect_url: %s", err))
	}

	// ------------- Optional query parameter "provider_id" -------------

	err = runtime.BindQueryParameter("form", true, false, "provider_id", ctx.QueryParams(), &params.ProviderId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter provider_id: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetLogin(ctx, params)
	return err
}

// GetLogout converts echo context to params.
func (w *ServerInterfaceWrapper) GetLogout(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetLogoutParams
	// ------------- Optional query parameter "redirect_url" -------------

	err = runtime.BindQueryParameter("form", true, false, "redirect_url", ctx.QueryParams(), &params.RedirectUrl)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter redirect_url: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetLogout(ctx, params)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET(baseURL+"/callback", wrapper.GetCallback)
	router.GET(baseURL+"/health", wrapper.GetHealth)
	router.GET(baseURL+"/login", wrapper.GetLogin)
	router.GET(baseURL+"/logout", wrapper.GetLogout)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/7xUTW/bOBD9K4M5K7Y3uSx0W+wC2QIBCrS9NUZAUyOJME0qw6ENN8h/L4ay0saxkbQI",
	"epQ4evP0PviALrQR6wdsKFl2g7gYsMab2LkArY+7Cj7+k6UHa7xfGbtOYEIDJktPQZw1+gG0keEThXXG",
	"CsWJJ6yxPENnhHZmjxVuidMIvv0LHyuMAwUzOKzxaraYXWKFg5E+KZf5tEwfOpKX/JRSZPdtXG9jQ4Xs",
	"E0ss+FyOPzRY4zXJvz/OBsNmQ0KcsP6qGmCN95lYeQazUfqKiRUy3WfH1GAtnKnCZHvaGCUk+0HnkrAL",
	"HT4+Vqdxkhh5HyBKqt/dBHgeYKnb0hBDoiLn1eLypYJfegKJawqwMwlyogYkgrGFJUhPYKyllA5Darq+",
	"1N+gJOASMDWOyQo10GaWnljJo5hOVcXnEcGlns17Ml76s7b+X45tT3YNFJohuiCzU16Og3j0o5eLxekf",
	"TcRbZ6mwziG40MHACur3s0JaB6Y0HCVta5w3K08Qg99Dy3EDOye9C0WP9d8JrM9JiLHCzB5r7EWGej5n",
	"LcDFoQAXKgeqNZM+BymWf3K7WuC12mcd+CyGJRXwMqg6lRg8VfykH+W+eFuxptjcKd3f6MHAcesa4jvX",
	"PPvcCW3SCZxqemGYzf5X6pET8VHQJRZpxvBMCpmOZq9H38cuZjkr/E3sCnRZGrNAbA96w380UGg0tHG0",
	"3cbQui6PFuigvjxYDbdBepfAmgBMKXuBQ1gKsvEpwooUzMeuo2bade3Em9XsNpyxV7m/r7/Lt7a3ENdb",
	"KuVyJbXZ+/1P9F/R/qhgY0vmZnDzqRXfAwAA//+0HyOKBgcAAA==",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
