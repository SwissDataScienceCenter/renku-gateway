package login

import (
	"fmt"
	"log/slog"
	"net/http"

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

	slog.Debug("LOGOUT", "redirectURL", redirectURL)

	session, err := l.sessions.Get(c)
	if err == nil {
		logoutSequence := l.getLogoutSequence(*session)
		slog.Debug("LOGOUT", "logoutSequence", logoutSequence)
		for i := range logoutSequence {
			providerID := logoutSequence[i]
			idToken, err := l.sessions.GetIDToken(c, *session, providerID)
			if err == nil {
				slog.Debug("LOGOUT", "idToken", idToken.String())
				handler, err := l.providerStore.EndSession(idToken, redirectURL, "foobar")
				if err != nil {
					return err
				}
				if handler != nil {
					return echo.WrapHandler(handler)(c)
				}
			}
		}

		// for providerID := range session.TokenIDs {
		// 	endSessionURL, err := l.providerStore.CheckEndSession(providerID)
		// 	if err == nil {
		// 		slog.Debug("LOGOUT", "providerID", providerID, "end session URL", endSessionURL)
		// 	}
		// }
		// for providerID := range session.TokenIDs {
		// 	idToken, err := l.sessions.GetIDToken(c, *session, providerID)
		// 	slog.Debug("LOGOUT", "providerID", providerID, "has token", (err == nil))
		// 	if err == nil {
		// 		slog.Debug("LOGOUT", "idToken", idToken.String())
		// 		handler, err := l.providerStore.EndSession(idToken, redirectURL, "")
		// 		if err != nil {
		// 			return err
		// 		}
		// 		err = echo.WrapHandler(handler)(c)
		// 		slog.Debug("LOGOUT", "handler error", err)
		// 		return err
		// 	}
		// }
		// l.sessions.GetIDToken(c, *session, "")
	}

	return fmt.Errorf("TODO")

	// // Delete the session from the store
	// err := l.sessions.Delete(c)
	// if err != nil {
	// 	return err
	// }
	// return c.Redirect(http.StatusFound, redirectURL)
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

// getLogoutSequence returns the logout sequence for a given session, with "gitlab" being last.
func (*LoginServer) getLogoutSequence(session models.Session) []string {
	const gitlab string = "gitlab"
	logoutSequence := make([]string, 0, len(session.TokenIDs))
	for providerID := range session.TokenIDs {
		if providerID != gitlab {
			logoutSequence = append(logoutSequence, providerID)
		}
	}
	_, hasGitlab := session.TokenIDs[gitlab]
	if hasGitlab {
		logoutSequence = append(logoutSequence, gitlab)
	}
	return logoutSequence
}
