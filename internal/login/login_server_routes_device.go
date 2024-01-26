package login

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
)

const deviceLoginSessionIDQueryParam = "sessionID"
const deviceLoginPath = "/api/auth/device/login"
const deviceLoginProviderID = "renkuCli"

// Called by the CLI to initialize the login flow session, the response is then showed to the user
func (l *LoginServer) DeviceLoginSession(c echo.Context) error {
	// a session handler middleware has set everything up here
	// a session cookie is there
	// and a session is already in the context
	session, ok := c.Get(models.DevicesSessionCtxKey).(models.Session)
	// NOTE: The renkuCLi provider is excluded here because the flow has already been started
	// the flow will be finished when the user visits the last page in the RedirectURL
	session.SetProviders(c.Request().Context(), "gitlab")
	if !ok {
		return gwerrors.ErrSessionParse
	}
	deviceAuthRes, err := l.providerStore.StartDeviceFlow(c.Request().Context(), deviceLoginProviderID) 
	if err != nil {
		return err
	}
	session.SetRedirectURL(c.Request().Context(), deviceAuthRes.VerificationURIComplete)
	modifiedRes := *deviceAuthRes
	// Swap the verification URL to lead to the gateway where the user will be made to log into
	// all other non-device flow providers and then finally into Keycloak
	newVerificationULR := l.config.RenkuBaseURL.JoinPath(deviceLoginPath)
	newVerificationULR.Query().Add(deviceLoginSessionIDQueryParam, session.ID)
	modifiedRes.VerificationURI = newVerificationULR.String() 
	modifiedRes.VerificationURIComplete = newVerificationULR.String() 
	return c.JSON(http.StatusOK, modifiedRes)
}

// The CLI tells to user to go here to start the login flow
func (l *LoginServer) DeviceLoginStart(c echo.Context) error {
	sessionID := c.QueryParam(deviceLoginSessionIDQueryParam)
	if sessionID == "" {
		return c.String(http.StatusUnprocessableEntity, "a session ID is required to proceed")
	}
	session, err := l.sessionStore.GetSession(c.Request().Context(), sessionID)
	if err != nil {
		return err
	}
	// pass this on to oauthNext
	slog.Info("session", "data", session)
	return nil	
}

type deviceCredentials struct {
	ProviderID string `json:"provider_id"`
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken string `json:"id_token"`
	ExpiresIn int `json:"expires_in"`
}

func (l *LoginServer) DeviceLoginTokenPost(c echo.Context) error {
	// the CLI posts the keycloak device login tokens here
	// use middleware to load the session in the context
	session, ok := c.Get(models.DevicesSessionCtxKey).(models.Session)
	if !ok {
		return gwerrors.ErrSessionParse
	}
	credentials := new(deviceCredentials)
	err := c.Bind(credentials)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("bad request: %s", err.Error()))
	}
	if credentials == nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("no credentials were passed"))
	}
	tokens, err := l.providerStore.VerifyTokens(c.Request().Context(), credentials.ProviderID, credentials.AccessToken, credentials.RefreshToken, credentials.IDToken)
	err = session.SaveTokens(c.Request().Context(), tokens[0], tokens[1], tokens[2], "")
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

