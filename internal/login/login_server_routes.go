package login

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

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
	handler, err := l.providerStore.CodeExchangeHandler(providerID)
	if err != nil {
		return err
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
			if session.UserID != "" {
				tokenID := "renku:" + session.UserID
				tokenSet.AccessToken.ID = tokenID
				tokenSet.RefreshToken.ID = tokenID
				tokenSet.IDToken.ID = tokenID
			}
		} else if providerID == "gitlab" && session.UserID != "" {
			tokenID := "gitlab:" + session.UserID
			tokenSet.AccessToken.ID = tokenID
			tokenSet.RefreshToken.ID = tokenID
			tokenSet.IDToken.ID = tokenID
		}
		return l.sessions.SaveTokens(c, session, tokenSet)
	}
	// Exchange the authorization code for credentials
	err = echo.WrapHandler(handler(tokenCallback))(c)
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

	session, err := l.sessions.Get(c)
	var renkuIdToken string = ""
	if err == nil {
		idToken, err := l.sessions.GetIDToken(c, *session, "renku")
		if err == nil {
			renkuIdToken = idToken.Value
		}
	}

	// Delete the session from the store
	err = l.sessions.Delete(c)
	if err != nil {
		return err
	}

	templateProviders := make(map[string]any, len(l.providerStore))
	for providerID, provider := range l.config.Providers {
		if providerID == "renku" && renkuIdToken != "" {
			logoutURL, err := url.Parse(provider.Issuer)
			if err != nil {
				return err
			}
			logoutURL = logoutURL.JoinPath("./protocol/openid-connect/logout")
			q := logoutURL.Query()
			q.Add("id_token_hint", renkuIdToken)
			logoutURL.RawQuery = q.Encode()
			templateProviders[providerID] = map[string]string{
				"logoutURL": logoutURL.String(),
			}
		}
		if l.config.LogoutGitLabUponRenkuLogout && providerID == "gitlab" {
			logoutURL := l.config.RenkuBaseURL.JoinPath(l.config.LoginRoutesBasePath).JoinPath("./gitlab/logout")
			if l.config.OldGitLabLogout {
				logoutURL, err = url.Parse(provider.Issuer)
				if err != nil {
					return err
				}
				logoutURL = logoutURL.JoinPath("./users/sign_out")
			}
			templateProviders[providerID] = map[string]string{
				"logoutURL": logoutURL.String(),
			}
		}
	}
	templateData := map[string]any{
		"renkuBaseURL": l.config.RenkuBaseURL,
		"redirectURL":  redirectURL,
		"providers":    templateProviders,
	}
	return c.Render(http.StatusOK, "logout", templateData)
}

func (l *LoginServer) GetGitLabToken(c echo.Context) error {
	userID := ""
	// Get the user id from the current session
	if userID == "" {
		session, err := l.sessions.Get(c)
		if err == nil {
			userID = session.UserID
		}
	}
	if userID == "" {
		return c.String(401, "Unauthorized")
	}
	gilabTokenID := "gitlab:" + userID
	gitlabAccessToken, err := l.tokenStore.GetFreshAccessToken(c.Request().Context(), gilabTokenID)
	if err != nil {
		return err
	}
	return c.JSON(200, map[string]any{
		"access_token": gitlabAccessToken.Value,
		"expires_at":   gitlabAccessToken.ExpiresAt.Unix(),
	})
}

func (l *LoginServer) GetGitLabLogout(c echo.Context) error {
	provider, ok := l.config.Providers["gitlab"]
	if !ok {
		return c.NoContent(404)
	}
	logoutURL := fmt.Sprintf("%s/users/sign_out", provider.Issuer)
	templateData := map[string]any{
		"logoutURL": logoutURL,
	}
	return c.Render(http.StatusOK, "gitlab_logout", templateData)
}

func (l *LoginServer) GetUserProfile(c echo.Context) error {
	redirectURL, err := l.providerStore.UserProfileURL("renku")
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, redirectURL.String())
}

func (*LoginServer) GetHealth(c echo.Context) error {
	return c.NoContent(http.StatusOK)
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
