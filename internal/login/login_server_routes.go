package login

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
)

// GetLogin is a handler for the initiation of a authorization code flow login for Renku
func (l *LoginServer) GetLogin(c echo.Context, params GetLoginParams) error {
	session, err := l.sessions.Create(c)
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
		loginSequence = defaultLoginSequence[:]
	}
	session.LoginSequence = loginSequence
	return l.nextAuthStep(c, session)
}

func (l *LoginServer) GetCallback(c echo.Context, params GetCallbackParams) error {
	// Load both the regular and the cli session (if present), see which fits
	state := c.Request().URL.Query().Get("state")
	if state == "" {
		return fmt.Errorf("a state parameter is required")
	}
	session, err := l.sessions.Get(c)
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
	tokenCallback := func(tokenSet sessions.AuthTokenSet) error {
		// Clear the state value before saving the tokens
		session.LoginState = ""
		// Make the token set and set the tokens' session ID
		tokenSet.AccessToken.SessionID = session.ID
		tokenSet.RefreshToken.SessionID = session.ID
		tokenSet.IDToken.SessionID = session.ID
		if providerID == "renku" {
			session.UserID = tokenSet.IDToken.Subject
		} else if providerID == "gitlab" && session.UserID != "" {
			tokenID := "gitlab:" + session.UserID
			tokenSet.AccessToken.ID = tokenID
			tokenSet.RefreshToken.ID = tokenID
			tokenSet.IDToken.ID = tokenID
		}
		return l.sessions.SaveTokens(c, session, tokenSet)
	}
	// Exchange the authorization code for credentials
	err = echo.WrapHandler(provider.CodeExchangeHandler(tokenCallback))(c)
	if err != nil {
		slog.Error("code exchange handler failed", "error", err, "requestID", utils.GetRequestID(c))
		return err
	}
	// Continue to the next authentication step
	return l.nextAuthStep(c, session)
}

func (l *LoginServer) GetLogout(c echo.Context, params GetLogoutParams) error {
	// Check redirect parameters
	var redirectURL string
	if params.RedirectUrl != nil && *params.RedirectUrl != "" {
		redirectURL = *params.RedirectUrl
	} else {
		redirectURL = l.config.RenkuBaseURL.String()
	}
	// Delete the session from the store
	err := l.sessions.Delete(c)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, redirectURL)
}

func (l *LoginServer) GetGitLabToken(c echo.Context) error {
	userID := ""
	// Get the user id from the authorization token
	accessToken := c.Request().Header.Get(echo.HeaderAuthorization)
	slog.Debug("LOGIN SERVER", "message", "gitlab token exchange", "accessToken", accessToken, "requestID", utils.GetRequestID(c))
	accessToken = strings.TrimPrefix(accessToken, "Bearer ")
	accessToken = strings.TrimPrefix(accessToken, "bearer ")
	if accessToken != "" {
		claims, err := l.providerStore.VerifyAccessToken(c.Request().Context(), "renku", accessToken)
		slog.Debug("LOGIN SERVER", "message", "gitlab token exchange", "verify", err, "requestID", utils.GetRequestID(c))
		if err == nil {
			slog.Debug("LOGIN SERVER", "message", "gitlab token exchange", "verify", claims.Subject, "requestID", utils.GetRequestID(c))
			userID = claims.Subject
		}
	}
	// Get the user id from the current session
	if userID == "" {
		session, err := l.sessions.Get(c)
		if err == nil {
			userID = session.UserID
		}
	}

	slog.Debug("LOGIN SERVER", "message", "gitlab token exchange", "userID", userID, "requestID", utils.GetRequestID(c))
	if userID == "" {
		return c.String(401, "Unauthorized")
	}

	gilabTokenID := "gitlab:" + userID
	slog.Debug("LOGIN SERVER", "message", "gitlab token exchange", "gilabTokenID", gilabTokenID, "requestID", utils.GetRequestID(c))
	gitlabAccessToken, err := l.tokenStore.GetFreshAccessToken(c.Request().Context(), gilabTokenID)
	slog.Debug("LOGIN SERVER", "message", "gitlab token exchange", "GetFreshAccessToken", err, "requestID", utils.GetRequestID(c))
	if err != nil {
		return err
	}
	return c.JSON(200, map[string]any{
		"access_token": gitlabAccessToken.Value,
		"expires_at":   gitlabAccessToken.ExpiresAt.Unix(),
	})
}

func (*LoginServer) GetHealth(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func (l *LoginServer) GetAuthTest(c echo.Context) error {
	session, err := l.sessions.Get(c)
	if err != nil {
		return err
	}
	return c.JSON(200, session)
}

// nextAuthStep sets up the beginning of the oauth flow and ends with
// the redirect of the user to the Provider's login and authorization page.
// Adapted from oauth2-proxy code.
func (l *LoginServer) nextAuthStep(
	c echo.Context,
	session *models.Session,
) error {
	// Get the next provider to authenticate with
	if session.LoginSequence == nil || len(session.LoginSequence) == 0 {
		// no more providers to login with, go to the application
		url := session.LoginRedirectURL
		session.LoginRedirectURL = ""
		if url == "" {
			url = l.config.RenkuBaseURL.String()
		}
		slog.Info("login completed", "requestID", utils.GetRequestID(c), "appRedirectURL", url)
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
		slog.Error("auth handler failed", "error", err, "requestID", utils.GetRequestID(c))
		return err
	}
	return echo.WrapHandler(handler)(c)
}
