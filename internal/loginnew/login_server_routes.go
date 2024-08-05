package loginnew

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/login"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/labstack/echo/v4"
)

// GetLogin is a handler for the initiation of a authorization code flow login for Renku
func (l *LoginServer2) GetLogin(c echo.Context, params login.GetLoginParams) error {
	session, err := l.sessionHandler.GetOrCreate(c)
	if err != nil {
		return err
	}
	// Check redirect parameters
	var appRedirectURL string
	if params.RedirectUrl != nil && *params.RedirectUrl != "" {
		appRedirectURL = *params.RedirectUrl
	} else {
		appRedirectURL = l.config.RenkuBaseURL.String()
	}
	session.LoginRedirectURL = appRedirectURL
	// Check provider IDs requested for login
	var loginSequence models.SerializableStringSlice
	if params.ProviderId != nil && len(*params.ProviderId) > 0 {
		loginSequence = *params.ProviderId
	} else {
		// TODO: Configure this
		loginSequence = []string{"renku", "gitlab"}
	}
	session.LoginSequence = loginSequence
	return l.nextAuthStep(c, session)
}

// nextAuthStep sets up the beginning of the oauth flow and ends with
// the redirect of the user to the Provider's login and authorization page.
// Adapted from oauth2-proxy code.
func (l *LoginServer2) nextAuthStep(
	c echo.Context,
	session sessions.Session,
) error {
	// Get the next provider to authenticate with
	if session.LoginSequence == nil || len(session.LoginSequence) == 0 {
		// no more providers to login with, go to the application
		url := session.LoginRedirectURL
		session.LoginRedirectURL = ""
		if url == "" {
			url = l.config.RenkuBaseURL.String()
		}
		slog.Info("login completed", "requestID", c.Request().Header.Get("X-Request-ID"), "appRedirectURL", url)
		return c.Redirect(http.StatusFound, url)
	}
	providerID := session.LoginSequence[0]
	// Setup the next login step
	err := session.GenerateLoginState()
	if err != nil {
		return err
	}
	// Handle the login
	handler, err := l.providerStore.AuthHandler(providerID, session.LoginState)
	if err != nil {
		slog.Error("auth handler failed", "error", err, "requestID", c.Request().Header.Get("X-Request-ID"))
		return err
	}
	err = echo.WrapHandler(handler)(c)
	return err
}

func (l *LoginServer2) GetCallback(c echo.Context, params login.GetCallbackParams) error {
	// Load both the regular and the cli session (if present), see which fits
	state := c.Request().URL.Query().Get("state")
	if state == "" {
		return fmt.Errorf("a state parameter is required")
	}
	session, err := l.sessionHandler.Get(c)
	if err != nil {
		return err
	}
	sessionState := session.LoginState
	if state != sessionState {
		return fmt.Errorf("state cannot be found in the existing session")
	}
	// Load the provider from the session
	if len(session.LoginSequence) == 0 {
		return fmt.Errorf("login sequence is invalid")
	}
	providerID := session.LoginSequence[0]
	session.LoginSequence = session.LoginSequence[1:]
	provider, found := l.providerStore[providerID]
	if !found {
		return fmt.Errorf("provider not found %s", providerID)
	}
	tokenCallback := func(accessToken, refreshToken, idToken models.AuthToken) error {
		// TODO
		// Update the session's token IDs
		session.TokenIDs[providerID] = accessToken.ID
		// Clear the state value
		session.LoginState = ""
		return nil
	}
	// Exchange the authorization code for credentials
	err = echo.WrapHandler(provider.CodeExchangeHandler(tokenCallback))(c)
	if err != nil {
		slog.Error("code exchange handler failed", "error", err, "requestID", c.Request().Header.Get("X-Request-ID"))
		return err
	}
	// Continue to the next authentication step
	return l.nextAuthStep(c, session)
}
