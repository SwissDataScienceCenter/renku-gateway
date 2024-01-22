package login

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
)

// GetLogin is a handler for the initiation of a authorization code flow login for Renku
func (l *LoginServer) GetLogin(c echo.Context, params GetLoginParams) error {
	session, ok := c.Get(config.SessionCtxKey).(models.Session)
	if !ok {
		return gwerrors.ErrSessionParse
	}
	var appRedirectURL string
	var providerIDs models.SerializableStringSlice
	// Check redirect parameters
	if params.RedirectUrl != nil && *params.RedirectUrl != "" {
		appRedirectURL = *params.RedirectUrl
	} else {
		appRedirectURL = l.config.DefaultAppRedirectURL
	}
	// Check provider IDs requested for login
	if params.ProviderId != nil && len(*params.ProviderId) > 0 {
		providerIDs = *params.ProviderId
	} else {
		providerIDs = l.config.DefaultProviderIDs()
	}
	// Set the providers and redirect
	// TODO: check if the session already has logged in with the provider
	err := session.SetRedirectURL(c.Request().Context(), appRedirectURL)
	if err != nil {
		return err
	}
	err = session.SetProviders(c.Request().Context(), providerIDs...)
	if err != nil {
		return err
	}
	return l.oAuthNext(c, session)
}

// GetDeviceLogin is a handler for the initiation of a device login for Renku, used by the CLI
func (l *LoginServer) GetDeviceLogin(c echo.Context, params GetDeviceLoginParams) error {
	session, ok := c.Get(config.SessionCtxKey).(models.Session)
	if !ok {
		return gwerrors.ErrSessionParse
	}
	var appRedirectURL string
	var providerIDs models.SerializableStringSlice
	if params.OriginalVerificationUriComplete != nil {
		appRedirectURL = *params.OriginalVerificationUriComplete
	}
	if params.OriginalVerificationUri != nil {
		appRedirectURL = *params.OriginalVerificationUri
	}
	if params.ProviderId != nil && len(*params.ProviderId) > 0 {
		providerIDs = *params.ProviderId
	} else {
		providerIDs = l.config.DefaultProviderIDs()
	}
	// Set the providers and redirect
	// TODO: check if the session already has logged in with the provider
	err := session.SetRedirectURL(c.Request().Context(), appRedirectURL)
	if err != nil {
		return err
	}
	err = session.SetProviders(c.Request().Context(), providerIDs...)
	if err != nil {
		return err
	}
	return l.oAuthNext(c, session)
}

// oauthNext sets up the beginning of the oauth flow and ends with
// the redirect of the user to the Provider's login and authorization page.
// Adapted from oauth2-proxy code.
func (l *LoginServer) oAuthNext(
	c echo.Context,
	session models.Session,
) error {
	// Get the providerID to login with
	providerID := session.PeekProviderID()
	if providerID == "" {
		// no more providers to login with go to the application
		url := session.PopRedirectURL()
		if url == "" {
			url = l.config.DefaultAppRedirectURL
		}
		slog.Info("login completed", "requestID", c.Request().Header.Get("X-Request-ID"), "appRedirectURL", url)
		return c.Redirect(http.StatusFound, url)
	}
	state := session.PeekOauthState()
	// Handle the login
	handler, err := l.providerStore.AuthHandler(providerID, state)
	if err != nil {
		slog.Error("auth handler failed", "error", err, "requestID", c.Request().Header.Get("X-Request-ID"))
		return err
	}
	err = echo.WrapHandler(handler)(c)
	return err
}

func (l *LoginServer) GetCallback(c echo.Context, params GetCallbackParams) error {
	session, ok := c.Get(config.SessionCtxKey).(models.Session)
	if !ok {
		return gwerrors.ErrSessionParse
	}
	state := c.Request().URL.Query().Get("state")
	if state == "" {
		return fmt.Errorf("a state parameter is required")
	}
	providerID := session.PeekProviderID()
	provider, found := l.providerStore[providerID]
	if !found {
		return fmt.Errorf("provider not found %s", providerID)
	}
	tokenCallback := func(accessToken, refreshToken models.OauthToken) error {
		return session.SaveTokens(c.Request().Context(), accessToken, refreshToken, state)
	}
	err := echo.WrapHandler(provider.CodeExchangeHandler(tokenCallback))(c)
	if err != nil {
		slog.Error("code exchange handler failed", "error", err, "requestID", c.Request().Header.Get("X-Request-ID"))
		return err
	}

	return l.oAuthNext(c, session)
}

// GetLogout logs the user out of the current session, removing the session cookie and removing the session
// in the session store.
func (l *LoginServer) GetLogout(c echo.Context, params GetLogoutParams) error {
	session, ok := c.Get(config.SessionCtxKey).(models.Session)
	if !ok {
		return gwerrors.ErrSessionParse
	}
	// figure out redirectURL
	var redirectURL = l.config.DefaultAppRedirectURL
	if params.RedirectUrl != nil {
		redirectURL = *params.RedirectUrl
	}
	// remove the session
	err := session.Remove(c.Request().Context())
	if err != nil {
		return err
	}
	// remove the cookie
	cookieName := l.sessionHandler.Cookie(&session).Name
	c.SetCookie(&http.Cookie{Name: cookieName, Value: "", MaxAge: -1})
	// redirect
	return c.Redirect(http.StatusFound, redirectURL)
}

func (*LoginServer) PostDeviceToken(c echo.Context) error {
	return c.String(http.StatusOK, "Coming soon")
	// just proxy to keycloak
}

func (l *LoginServer) PostDevice(c echo.Context) error {
	return c.String(http.StatusOK, "Coming soon")
	// just proxy to keycloak
}

func (*LoginServer) GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "Running")
}

func (*LoginServer) PostLogout(c echo.Context) error {
	logoutToken := c.FormValue("logout_token")
	type invalidLogoutResponse struct {
		Error            string `json:"error,omitempty"`
		ErrorDescription string `json:"error_description,omitempty"`
	}
	if logoutToken == "" {
		return c.JSON(http.StatusBadRequest, invalidLogoutResponse{Error: "A logout token has to be provided"})
	}
	// TODO: validate logout token (see https://openid.net/specs/openid-connect-backchannel-1_0.html#Validation)
	// TODO: remove session in redis
	return c.NoContent(http.StatusOK)
}
