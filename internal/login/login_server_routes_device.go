package login

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"net/http"
// 	"net/url"

// 	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
// 	"github.com/labstack/echo/v4"
// 	"github.com/labstack/echo/v4/middleware"
// 	"github.com/zitadel/oidc/v2/pkg/oidc"
// )

// const cliLoginSessionIDQueryParam = "session_id"
// const cliLoginProviderID = "renkucli"

// // Called by the CLI to initialize the login flow session, the response is then showed to the user
// func (l *LoginServer) PostDeviceLogin(c echo.Context) error {
// 	session, err := l.cliSessionHandler.Create(c)
// 	if err != nil {
// 		return err
// 	}
// 	// NOTE: The renkuCLi provider is excluded here because the device flow has already been started for it.
// 	// The flow will be finished when the user visits the last page in the RedirectURL
// 	err = session.SetProviders(c.Request().Context(), "gitlab")
// 	if err != nil {
// 		return err
// 	}
// 	deviceAuthRes, err := l.providerStore.StartDeviceFlow(c.Request().Context(), cliLoginProviderID)
// 	if err != nil {
// 		return err
// 	}
// 	verificationURI := deviceAuthRes.VerificationURIComplete
// 	if verificationURI == "" {
// 		verificationURI = deviceAuthRes.VerificationURI
// 		if verificationURI == "" {
// 			return fmt.Errorf("did not receive a verification uri from the cli provider")
// 		}
// 	}
// 	err = session.SetRedirectURL(c.Request().Context(), verificationURI)
// 	if err != nil {
// 		return err
// 	}
// 	modifiedRes := *deviceAuthRes
// 	// Swap the verification URL to lead to the gateway where the user will be made to log into
// 	// all other non-device flow providers and then finally into Keycloak - this leads to GetDeviceLogin handler below
// 	newVerificationURL := l.config.RenkuBaseURL.JoinPath(l.config.EndpointsBasePath, "device/login")
// 	vals := newVerificationURL.Query()
// 	vals.Add(cliLoginSessionIDQueryParam, session.ID)
// 	newVerificationURL.RawQuery = vals.Encode()
// 	modifiedRes.VerificationURI = newVerificationURL.String()
// 	modifiedRes.VerificationURIComplete = newVerificationURL.String()
// 	c.SetCookie(l.cliSessionHandler.Cookie(&session))
// 	c.Response().Header().Set(models.CliSessionHeaderKey, session.ID)
// 	return c.JSON(http.StatusOK, modifiedRes)
// }

// // The CLI tells to user to go here to start the login flow
// func (l *LoginServer) GetDeviceLogin(c echo.Context, params GetDeviceLoginParams) error {
// 	if params.SessionId == nil || *params.SessionId == "" {
// 		return c.String(http.StatusUnprocessableEntity, "a session ID is required to proceed")
// 	}
// 	session, err := l.sessionStore.GetSession(c.Request().Context(), *params.SessionId)
// 	if err != nil {
// 		slog.Error("DEVICE LOGIN", "error", err, "sessionID", *params.SessionId)
// 		return err
// 	}
// 	// NOTE: You have to use a cookie here because setting a session ID in the headers would not
// 	// "survive" all the redirects that happen during the oauth login flows
// 	cookie := l.cliSessionHandler.Cookie(&session)
// 	c.SetCookie(cookie)
// 	return l.oAuthNext(c, session)
// }

// // PostDeviceToken is just here to satisfy the requirements of the code generation from the openapi
// // spec. If this is removed then the endpoint has to be removed from the openapi spec and it is better
// // to have information about this in the spec. This endpoint is never used because it is always
// // proxied to Keycloak and handled there.
// func (l *LoginServer) PostDeviceToken(c echo.Context) error {
// 	return fmt.Errorf("this should never be reached beacuse it is proxied to Keycloak")
// }

// // Destroy the session and log the user user out of all the places where they logged in
// func (l *LoginServer) PostDeviceLogout(c echo.Context) error {
// 	err := l.cliSessionHandler.Remove(c)
// 	if err != nil {
// 		return err
// 	}
// 	return c.String(http.StatusOK, "need to implement logout")
// }

// // Middleware that proxies the request from the CLI to get the access tokens from Keycloak from the
// // device flow. It will copy the response and save it before passing it back on to the CLI.
// func (l *LoginServer) DeviceTokenProxy() ([]echo.MiddlewareFunc, error) {
// 	replaceBody := func(r *http.Response, newBody []byte) error {
// 		modifiedOutput := bytes.NewBuffer(newBody)
// 		r.Body = io.NopCloser(modifiedOutput)
// 		r.ContentLength = int64(modifiedOutput.Len())
// 		// If the content length response header is not set a lot of browser requests and even curl will fail
// 		r.Header.Set(echo.HeaderContentLength, fmt.Sprintf("%d", modifiedOutput.Len()))
// 		return nil
// 	}
// 	responseHandler := func(r *http.Response) error {
// 		// check the response from Keycloak and if it is not 200 return the whole body unchanged
// 		if r.StatusCode < 200 || r.StatusCode >= 400 {
// 			slog.Info("LOGIN", "message", "skipping response rewrite", "statusCode", r.StatusCode)
// 			return nil
// 		}
// 		sessionID := r.Request.Header.Get(models.CliSessionHeaderKey)
// 		if sessionID == "" {
// 			sessionCookie, err := r.Request.Cookie(models.CliSessionCookieName)
// 			if err != nil {
// 				return err
// 			}
// 			sessionID = sessionCookie.Value
// 		}
// 		session, err := l.sessionStore.GetSession(r.Request.Context(), sessionID)
// 		if err != nil {
// 			return err
// 		}
// 		// decode response and save credentials
// 		credentials := new(oidc.AccessTokenResponse)
// 		err = json.NewDecoder(r.Body).Decode(credentials)
// 		if err != nil {
// 			return err
// 		}
// 		parsedTokens, err := l.providerStore.VerifyTokens(r.Request.Context(), cliLoginProviderID, credentials.AccessToken, credentials.RefreshToken, credentials.IDToken)
// 		if err != nil {
// 			return err
// 		}
// 		err = session.SaveTokens(r.Request.Context(), parsedTokens[0], parsedTokens[1], parsedTokens[2], "")
// 		if err != nil {
// 			return err
// 		}
// 		// remove credentials from response
// 		modifiedCredentials := *credentials
// 		modifiedCredentials.AccessToken = "redacted"
// 		modifiedCredentials.RefreshToken = "redacted"
// 		modifiedCredentials.IDToken = "redacted"
// 		modifiedCredentialsBytes, err := json.Marshal(modifiedCredentials)
// 		if err != nil {
// 			return err
// 		}
// 		return replaceBody(r, modifiedCredentialsBytes)
// 	}
// 	setHost := func(host string) echo.MiddlewareFunc {
// 		return func(next echo.HandlerFunc) echo.HandlerFunc {
// 			return func(c echo.Context) error {
// 				c.Request().Host = host
// 				return next(c)
// 			}
// 		}
// 	}
// 	providerConfig, found := l.config.Providers[cliLoginProviderID]
// 	if !found {
// 		return nil, fmt.Errorf("cannot find the %s provider in the configuration", cliLoginProviderID)
// 	}
// 	if providerConfig.Issuer == "" {
// 		return nil, fmt.Errorf("the OIDC provider cannot have a blank issuer field")
// 	}
// 	issuerURL, err := url.Parse(providerConfig.Issuer)
// 	if err != nil {
// 		return nil, fmt.Errorf("cannot parse the issuer URL of the OIDC config")
// 	}
// 	// NOTE: the proxying will fail if the url that is being proxied to does not have a blank path,
// 	// this is because there is another middleware that handles the path rewrititng before the proxy kicks in
// 	issuerURLBlankPath := *issuerURL
// 	issuerURLBlankPath.Path = ""
// 	issuerURLBlankPath.RawPath = ""
// 	proxy := middleware.ProxyWithConfig(middleware.ProxyConfig{
// 		ModifyResponse: responseHandler,
// 		Balancer: middleware.NewRandomBalancer([]*middleware.ProxyTarget{
// 			{URL: &issuerURLBlankPath},
// 		}),
// 	})
// 	gwDeviceTokenURL := l.config.RenkuBaseURL.JoinPath(l.config.EndpointsBasePath, "/device/token")
// 	pathRewrite := middleware.Rewrite(map[string]string{
// 		// when the CLI keeps checking the token endpoint to retrieve the Keycloak access token it will be simply routed to Keycloak
// 		gwDeviceTokenURL.Path: issuerURL.JoinPath("protocol/openid-connect/token").String(),
// 	})
// 	return []echo.MiddlewareFunc{pathRewrite, setHost(issuerURL.Host), proxy}, nil
// }
