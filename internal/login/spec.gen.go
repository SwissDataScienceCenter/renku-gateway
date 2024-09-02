// Package login provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.16.2 DO NOT EDIT.
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
	Code  string `form:"code" json:"code"`
	State string `form:"state" json:"state"`
}

// GetLoginParams defines parameters for GetLogin.
type GetLoginParams struct {
	RedirectUrl *string `form:"redirect_url,omitempty" json:"redirect_url,omitempty"`

	// ProviderId Providing `provider_id` query parameters should be used for testing only.
	ProviderId *[]string `form:"provider_id,omitempty" json:"provider_id,omitempty"`
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

	// (GET /user-profile)
	GetUserProfile(ctx echo.Context) error
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

// GetUserProfile converts echo context to params.
func (w *ServerInterfaceWrapper) GetUserProfile(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetUserProfile(ctx)
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
	router.GET(baseURL+"/user-profile", wrapper.GetUserProfile)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/7xVwY7bNhD9lcFcetHa7uZS6Fa0wLZAgAZpe2qMhKZGEmGaVDhDG26w/14MJSW7Xhm7",
	"C7S9WeTwzeN7b+gv6EIbsf6CDbFNbhAXA9b4NnYuQOvjqYLffszSgzXe74zdM5jQgMnSUxBnjR6ANiZ4",
	"T2GfsUJx4glrLN9wZ4RO5owVHinxCH78Hu8rjAMFMzis8c1qs7rFCgcjPSuX9dxMPzqSp/yUUkzu77G9",
	"jQ0Vsl9ZYsFPZfvXBmu8I/np295gkjmQUGKs/1INsMbPmZLyDOag9BUTK0z0ObtEDdaSMlXItqeDUUJy",
	"HrSOJbnQ4f19tYzDYuR1QFst5iEGpqLGm83tUwH+6Akk7inAyTBkpgYkgrGlCUhPYKwl5qlIPdNFZUEs",
	"4BgSNS6RFWqgzUl6SnoJFNOpKPjYYdzq3ron46W/6sovZdv2ZPdAoRmiC7JasmIsxIuL3m42yxdlSkdn",
	"qbDOIbjQwZAU1J9XhbQWzGZeBOVonDc7TxCDP0Ob4gFOTnoXih77HxiszyyUsMKcPNbYiwz1ep00vzfd",
	"mN8blQPVmlmfSYrt/9ldLfA6mVcd+F1MEi7gpVB1KjH4OqGLfpRxf9lczLH5qHSfmYfH3N6leHSNmvdp",
	"KD8pfXTNJygN4Ftr4D5m38COxlwrdSEWPakyrj4o1SVuD2AfUXNCB17gWM0LJiVzfs3oZaZ0MUQSi+xj",
	"MGf1TUer58fKxy5muWrq29gV6NI0ZoHYTl7CzzRQaEZlSo2NoXVdHu3VQl2cYgQfgvSOwZoAiTh7gSmI",
	"Bdl4jrAjBfOx66iZe9058WY3yr4UHeX+72Zn+9KXoRDXF5Bzee7a7P35Af3ntVeEmyHF1um/1hUH3k/E",
	"eXZZT33HMJ0bfV7W50+m9G6C/8/TdXBs9V4Xj9L4sqzN4NbzS/JPAAAA///N5yMY+QcAAA==",
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
