package main

import (
	"fmt"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/commonconfig"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/errors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
)

const SessionIDCtxKey string = "sessionID"

func (l *LoginServer) GetLogin(c echo.Context, params GetLoginParams) error {
	var session models.Session
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
		providerIDs = l.config.DefaultProviderIDs
	}
	// Get the session from the context - the session middleware already got it from the store
	session, ok := c.Get(commonconfig.SessionCtxKey).(models.Session)
	if !ok {
		return errors.ErrSessionParse
	}
	session.SetProviderIDs(providerIDs)
	session.SetRedirectURL(appRedirectURL)

	return l.oAuthNext(c, session)
}

// oauthStart sets up the beginning of the oauth flow and ends with
// the redirect of the user to the Provider's login and authorization page.
// Adapted from oauth2-proxy code.
func (l *LoginServer) oAuthNext(
	c echo.Context,
	session models.Session,
) error {
	// Get the providerID to login with
	providerID := session.PeekProviderID()
	// Persist session in store
	err := l.sessionStore.SetSession(c.Request().Context(), session)
	if err != nil {
		return err
	}
	if providerID == "" {
		// no more providers to login with go to the application
		return c.Redirect(http.StatusFound, session.RedirectURL)
	}
	// Handle the login
	handler, err := l.providerStore.AuthHandler(providerID)
	if err != nil {
		return err
	}
	err = echo.WrapHandler(handler)(c)
	return err
}

func (l *LoginServer) GetCallback(c echo.Context) error {
	session, ok := c.Get(commonconfig.SessionCtxKey).(models.Session)
	if !ok {
		return fmt.Errorf("cannot cast session from context")
	}
	providerID := session.PopProviderID()
	provider, found := l.providerStore[providerID]
	if !found {
		return fmt.Errorf("provider not found %s", providerID)
	}
	err := echo.WrapHandler(provider.CodeExchangeHandler(func(accessToken, refreshToken models.OauthToken) error {
		session.AddTokenID(accessToken.ID)
		if err := l.tokenStore.SetAccessToken(c.Request().Context(), accessToken); err != nil {
			return err
		}
		return l.tokenStore.SetRefreshToken(c.Request().Context(), refreshToken)
	}))(c)
	if err != nil {
		return err
	}

	return l.oAuthNext(c, session)
}

// GetLogout logs the user out of the current session, removing the session cookie and removing the session
// in the session store.
func (l *LoginServer) GetLogout(c echo.Context, params GetLogoutParams) error {
	// figure out redirectURL
	var redirectURL = l.config.DefaultAppRedirectURL
	if params.RedirectUrl != nil {
		redirectURL = *params.RedirectUrl
	}
	// get session cookie
	cookie, err := c.Request().Cookie(commonconfig.SessionCookieName)
	if err == http.ErrNoCookie {
		return c.Redirect(http.StatusFound, redirectURL)
	}
	if err != nil {
		return err
	}
	// remove the session
	if cookie.Value != "" {
		if err := l.sessionStore.RemoveSession(c.Request().Context(), cookie.Value); err != nil {
			return err
		}
	}
	// remove the cookie
	c.SetCookie(&http.Cookie{Name: commonconfig.SessionCookieName, Value: "", MaxAge: -1})
	// redirect
	return c.Redirect(http.StatusFound, redirectURL)
}

func (*LoginServer) PostCliLoginComplete(c echo.Context) error {
	return c.String(http.StatusOK, "Coming soon")
}

func (*LoginServer) PostCliLoginInit(c echo.Context) error {
	return c.String(http.StatusOK, "Coming soon")
}

func (*LoginServer) GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "Running")
}

func (*LoginServer) PostBackchannelLogout(c echo.Context) error {
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
