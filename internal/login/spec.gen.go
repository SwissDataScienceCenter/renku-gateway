// Package login provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.12.4 DO NOT EDIT.
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

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

// GetCallbackParams defines parameters for GetCallback.
type GetCallbackParams struct {
	Code         string  `form:"code" json:"code"`
	State        string  `form:"state" json:"state"`
	SessionState *string `form:"session_state,omitempty" json:"session_state,omitempty"`
}

// GetDeviceLoginParams defines parameters for GetDeviceLogin.
type GetDeviceLoginParams struct {
	SessionId *string `form:"session_id,omitempty" json:"session_id,omitempty"`
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

	// (GET /device/login)
	GetDeviceLogin(ctx echo.Context, params GetDeviceLoginParams) error

	// (POST /device/login)
	PostDeviceLogin(ctx echo.Context) error

	// (POST /device/logout)
	PostDeviceLogout(ctx echo.Context) error

	// (POST /device/token)
	PostDeviceToken(ctx echo.Context) error

	// (GET /health)
	GetHealth(ctx echo.Context) error

	// (GET /login)
	GetLogin(ctx echo.Context, params GetLoginParams) error

	// (GET /logout)
	GetLogout(ctx echo.Context, params GetLogoutParams) error

	// (POST /logout)
	PostLogout(ctx echo.Context) error
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

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetCallback(ctx, params)
	return err
}

// GetDeviceLogin converts echo context to params.
func (w *ServerInterfaceWrapper) GetDeviceLogin(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetDeviceLoginParams
	// ------------- Optional query parameter "session_id" -------------

	err = runtime.BindQueryParameter("form", true, false, "session_id", ctx.QueryParams(), &params.SessionId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter session_id: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetDeviceLogin(ctx, params)
	return err
}

// PostDeviceLogin converts echo context to params.
func (w *ServerInterfaceWrapper) PostDeviceLogin(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostDeviceLogin(ctx)
	return err
}

// PostDeviceLogout converts echo context to params.
func (w *ServerInterfaceWrapper) PostDeviceLogout(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostDeviceLogout(ctx)
	return err
}

// PostDeviceToken converts echo context to params.
func (w *ServerInterfaceWrapper) PostDeviceToken(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostDeviceToken(ctx)
	return err
}

// GetHealth converts echo context to params.
func (w *ServerInterfaceWrapper) GetHealth(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
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

	// Invoke the callback with all the unmarshalled arguments
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

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetLogout(ctx, params)
	return err
}

// PostLogout converts echo context to params.
func (w *ServerInterfaceWrapper) PostLogout(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostLogout(ctx)
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
	router.GET(baseURL+"/device/login", wrapper.GetDeviceLogin)
	router.POST(baseURL+"/device/login", wrapper.PostDeviceLogin)
	router.POST(baseURL+"/device/logout", wrapper.PostDeviceLogout)
	router.POST(baseURL+"/device/token", wrapper.PostDeviceToken)
	router.GET(baseURL+"/health", wrapper.GetHealth)
	router.GET(baseURL+"/login", wrapper.GetLogin)
	router.GET(baseURL+"/logout", wrapper.GetLogout)
	router.POST(baseURL+"/logout", wrapper.PostLogout)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/7xWQW/jRg/9K8SctXa+7HcofGt3gW1QAy12t710g8VYoqSpx0NlSNn1BvnvBWckW0ns",
	"NCnSHq2hHznvkXxza0radBQwCJvF7V1hXKjJLG5NhVxG14mjYBZmSY0LUHvaFfDz9720UFrvV7ZcM9hQ",
	"ge2lxSCutPoHqCnCRwzr3hRGnHg0C5N+Q2MFd3ZvCrPFyBl8+z9zVxjqMNjOmYV5O7uYXZrCdFZarcrM",
	"x2T6o0F5XJ+WRNF9y+lLqjAVe6jSJPyYjq8qszAfUN4dzzob7QYFI5vF77fGKeRNj1HrDHaj5SumKUzE",
	"m95FrMxCYo+F4bLFjdWCZN9pHEt0oTF3d8VpHBYrrwOErPx9HQHPA1xrNu4oMCY6315cPmbwc4sgtMYA",
	"O8vQM1YgBLZMVYK0CLYskXkIUtH1o14DWcAxRKxcxFKwgrqP0mLU4o3YRlk1MfXDtX6aV7h1Jc69dtVZ",
	"TX9z7BRstU+ZesaoJbHYKOCpaVxowAUhqFxdY8Qg0EXaugojF1CT97Q7/j/nzG0xhoGVdIahmoEy8G55",
	"BYyh4mPKFvX+j/POTvXU+5QkTcvz2mpU0VWvI2Eq+b4YQvn+j0nSI0/5MlOpMlXm+q4wHfEJaT4pF5mj",
	"pKFCpubQwdevkrks0gAy/IT70pNdH3nUgOFPBWyocrXDDHiIHS+cei2L4mS8zbvl1WMBfiF+oMA90i4v",
	"Lk6TdshUR9pMim2tQElBrAuThkjbRWvaYnT1uPJ+/biEXe6VMXDnvIeGjiwLHcDPEH5/OKhP3J/WYEkN",
	"A/Uybe6hm+CNFsCg261sbQjoIcPBCkvb86RIxxBIewJZe8OFdLSKtNPTrKfjQql3DH/0LCP/UJFOgh4D",
	"BVhha30NVB+wn9ZHL/dcgTKdloH7tITq3vt9mkWslIO/JTMtrfNcjqO/RuwYOvL+cDMhaFCmC/Cgf8Q6",
	"Irf5w67FMBkIx1C7YL37dn+D6ZpxrMddpD9dHs9Dxw2Bg0lOluy0P9djtOIEwVhip3OexsRuscq1qLF7",
	"FHxqTD4nWp6rwtBk/vAWgNYyrHCSrFIn//85gOGf1nmsiql9MNiICpJoJ4Kb3pVrv4dhndwL21hfU9xo",
	"rnOqt2i9tGed5cd0XLZYrnX1d+SCnFznOfDZBDHGxJDu3z4EvUwXFdTv87xrwOgGDx4wW+u8XXkECn6f",
	"hd45aYd5XH/HUPqeJU1VH71ZmFakW8znyVffDC3zRh9iRu3ioe3+h8lVgaet/Un/SA/Fk3K8wFdH//uq",
	"5f6D19XokQ+N2Qlu+AROMX6wMdr96zh27p2RIdvg7OyD6mgWJ/leUnPc+GoDVA80w3vsMFTaqpTVLinU",
	"rukz8+M6HzfSl5CWYmmDLqXeHwwjIVvPBCtUsONuVogPTrxdzb6EM6pmL3hNWa//FWs5DtNpH/lhYrik",
	"03A5+O7JHfwCE/z0guIeLpo8sHPbufk4oH8FAAD//zOlQ4v3DQAA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
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
	var res = make(map[string]func() ([]byte, error))
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
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
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