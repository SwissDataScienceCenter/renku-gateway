package revproxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type AuthOption func(*Auth)

type TokenInjector func(c echo.Context, token models.AuthToken) error

func InjectInHeader(headerKey string) AuthOption {
	return func(a *Auth) {
		a.tokenInjector = func(c echo.Context, token models.AuthToken) error {
			existingToken := c.Request().Header.Get(headerKey)
			if existingToken != "" {
				slog.Debug(
					"PROXY AUTH MIDDLEWARE",
					"message",
					"token already present in header, skipping",
					"header",
					headerKey,
					"token",
					token.String(),
					"requestID",
					utils.GetRequestID(c),
				)
				return nil
			}
			slog.Debug(
				"PROXY AUTH MIDDLEWARE",
				"message",
				"injected token in header",
				"header",
				headerKey,
				"token",
				token.String(),
				"requestID",
				utils.GetRequestID(c),
			)
			c.Request().Header.Set(headerKey, token.Value)
			return nil
		}
	}
}

func InjectBearerToken() AuthOption {
	return func(a *Auth) {
		a.tokenInjector = func(c echo.Context, token models.AuthToken) error {
			existingToken := c.Request().Header.Get(echo.HeaderAuthorization)
			if existingToken != "" {
				slog.Debug(
					"PROXY AUTH MIDDLEWARE",
					"message",
					"token already present in header, skipping",
					"header",
					echo.HeaderAuthorization,
					"token",
					token.String(),
					"requestID",
					utils.GetRequestID(c),
				)
				return nil
			}
			slog.Debug(
				"PROXY AUTH MIDDLEWARE",
				"message",
				"injected token in header",
				"header",
				echo.HeaderAuthorization,
				"providerID",
				a.providerID,
				"token",
				token.String(),
				"requestID",
				utils.GetRequestID(c),
			)
			c.Request().Header.Set(echo.HeaderAuthorization, fmt.Sprintf("Bearer %s", token.Value))
			return nil
		}
	}
}

func WithTokenInjector(injector TokenInjector) AuthOption {
	return func(a *Auth) {
		a.tokenInjector = injector
	}
}

func WithProviderID(providerID string) AuthOption {
	return func(a *Auth) {
		a.providerID = providerID
	}
}

func WithTokenType(tokenType models.OauthTokenType) AuthOption {
	return func(a *Auth) {
		a.tokenType = tokenType
	}
}

func AuthWithSessionStore(sessions *sessions.SessionStore) AuthOption {
	return func(a *Auth) {
		a.sessions = sessions
	}
}

// Auth generates middleware that will inject tokens in the proxied http requests
type Auth struct {
	sessions      *sessions.SessionStore
	tokenInjector TokenInjector
	providerID    string
	tokenType     models.OauthTokenType
}

func NewAuth(options ...AuthOption) (Auth, error) {
	auth := Auth{providerID: "renku", tokenType: models.AccessTokenType}
	for _, opt := range options {
		opt(&auth)
	}
	if auth.sessions == nil {
		return Auth{}, fmt.Errorf("session store not initialized")
	}
	if auth.tokenInjector == nil {
		return Auth{}, fmt.Errorf("token injector not initialized")
	}
	if auth.tokenType != models.AccessTokenType && auth.tokenType != models.RefreshTokenType && auth.tokenType != models.IDTokenType {
		return Auth{}, fmt.Errorf("unknown token type in authentication middleware %s", auth.tokenType)
	}
	return auth, nil
}

func (a *Auth) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, err := a.sessions.Get(c)
			if err != nil {
				slog.Debug(
					"PROXY AUTH MIDDLEWARE",
					"message",
					"session not available, continuing with middleware chain",
					"providerID",
					a.providerID,
					"tokenType",
					a.tokenType,
					"requestID",
					utils.GetRequestID(c),
				)
				return next(c)
			}

			var token models.AuthToken
			if a.tokenType == models.AccessTokenType {
				token, err = a.sessions.GetAccessToken(c, *session, a.providerID)
			} else if a.tokenType == models.RefreshTokenType {
				token, err = a.sessions.GetRefreshToken(c, *session, a.providerID)
			} else if a.tokenType == models.IDTokenType {
				token, err = a.sessions.GetIDToken(c, *session, a.providerID)
			} else {
				return fmt.Errorf("unknown token type in authentication middleware %s", a.tokenType)
			}
			if err != nil {
				switch err {
				case gwerrors.ErrTokenNotFound:
					slog.Debug(
						"PROXY AUTH MIDDLEWARE",
						"message",
						"token not found, continuing with middleware chain",
						"sessionID",
						session.ID,
						"providerID",
						a.providerID,
						"tokenType",
						a.tokenType,
						"requestID",
						utils.GetRequestID(c),
					)
					return next(c)
				case gwerrors.ErrTokenExpired:
					slog.Debug(
						"PROXY AUTH MIDDLEWARE",
						"message",
						"token expired, continuing with middleware chain",
						"sessionID",
						session.ID,
						"providerID",
						a.providerID,
						"tokenType",
						a.tokenType,
						"requestID",
						utils.GetRequestID(c),
					)
					return next(c)
				default:
					slog.Info(
						"PROXY AUTH MIDDLEWARE",
						"message",
						"token could not be loaded, continuing with middleware chain",
						"error",
						err,
						"sessionID",
						session.ID,
						"providerID",
						a.providerID,
						"tokenType",
						a.tokenType,
						"requestID",
						utils.GetRequestID(c),
					)
					return next(c)
				}
			}
			err = a.tokenInjector(c, token)
			if err != nil {
				return err
			}
			return next(c)
		}
	}
}

var notebooksGitlabAccessTokenInjector TokenInjector = func(c echo.Context, accessToken models.AuthToken) error {
	headerKey := "Renku-Auth-Git-Credentials"
	existingToken := c.Request().Header.Get(headerKey)
	if existingToken != "" {
		slog.Debug(
			"PROXY AUTH MIDDLEWARE",
			"message",
			"token already present in header, skipping",
			"header",
			headerKey,
			"token",
			accessToken.String(),
			"requestID",
			utils.GetRequestID(c),
		)
		return nil
	}

	// NOTE: As long as the token comes from the database we can trust it and do not have to validate it.
	// Each service that the request ultimately goes to will also validate before it uses the token
	type gitCredentials struct {
		Provider             string `json:"provider"`
		AuthorizationHeader  string
		AccessTokenExpiresAt int64
	}
	output := map[string]gitCredentials{}
	credentials := gitCredentials{Provider: "GitLab", AuthorizationHeader: fmt.Sprintf("Bearer %s", accessToken.Value)}
	// NOTE: if the expiry date in the claims is absent we assume that the token does not expire
	var expiresAt int64 = -1
	if !accessToken.ExpiresAt.IsZero() {
		expiresAt = accessToken.ExpiresAt.Unix()
	}
	credentials.AccessTokenExpiresAt = expiresAt
	tokenURL, err := url.Parse(accessToken.TokenURL)
	if err != nil {
		return err
	}
	// The notebook service expects a specific format for the headers where we have to provide the
	// base URL for gitlab. That is why we strip out all the things from the token URL.
	issuerURL := tokenURL
	issuerURL.Fragment = ""
	issuerURL.RawFragment = ""
	issuerURL.Path = ""
	issuerURL.RawPath = ""
	issuerURL.RawQuery = ""
	output[issuerURL.String()] = credentials
	outputJson, err := json.Marshal(output)
	if err != nil {
		return err
	}
	headerVal := base64.StdEncoding.EncodeToString(outputJson)
	slog.Debug(
		"PROXY AUTH MIDDLEWARE",
		"message",
		"injected token in header",
		"header",
		headerKey,
		"token",
		accessToken.String(),
		"requestID",
		utils.GetRequestID(c),
	)
	c.Request().Header.Set(headerKey, headerVal)
	return nil
}

var coreSvcRenkuIdTokenInjector TokenInjector = func(c echo.Context, idToken models.AuthToken) error {
	headerKey := "Renku-User"
	existingToken := c.Request().Header.Get(headerKey)
	if existingToken != "" {
		slog.Debug(
			"PROXY AUTH MIDDLEWARE",
			"message",
			"token already present in header, skipping",
			"header",
			headerKey,
			"token",
			idToken.String(),
			"requestID",
			utils.GetRequestID(c),
		)
		return nil
	}

	var claims oidc.IDTokenClaims
	_, err := oidc.ParseToken(idToken.Value, &claims)
	if err != nil {
		return err
	}
	userId := claims.Subject
	email := claims.Email
	name := claims.Name

	slog.Debug(
		"PROXY AUTH MIDDLEWARE",
		"message",
		"injected token in header",
		"header",
		headerKey,
		"token",
		idToken.String(),
		"requestID",
		utils.GetRequestID(c),
	)
	c.Request().Header.Set(headerKey, idToken.Value)
	slog.Debug(
		"PROXY AUTH MIDDLEWARE",
		"message",
		"injected user info in header",
		"Renku-user-id",
		userId,
		"requestID",
		utils.GetRequestID(c),
	)
	c.Request().Header.Set("Renku-user-id", userId)
	c.Request().Header.Set("Renku-user-email", base64.StdEncoding.EncodeToString([]byte(email)))
	c.Request().Header.Set("Renku-user-fullname", base64.StdEncoding.EncodeToString([]byte(name)))
	return nil
}

// Sets up Basic Auth for Gitlab
var gitlabCliTokenInjector TokenInjector = func(c echo.Context, accessToken models.AuthToken) error {
	if accessToken.Value == "" {
		return nil
	}
	c.Request().SetBasicAuth("oauth2", accessToken.Value)
	return nil
}

// Sets the Gitlab-Access-Token and Gitlab-Access-Token-Expires-At values in the header, the expiry part
// is needed because Gitlab access tokens are opaque and the holder cannot easily tell when they expire.
var dataServiceGitlabAccessTokenInjector TokenInjector = func(c echo.Context, accessToken models.AuthToken) error {
	headerKey := "Gitlab-Access-Token"
	existingToken := c.Request().Header.Get(headerKey)
	if existingToken != "" {
		slog.Debug(
			"PROXY AUTH MIDDLEWARE",
			"message",
			"token already present in header, skipping",
			"header",
			headerKey,
			"token",
			accessToken.String(),
			"requestID",
			utils.GetRequestID(c),
		)
		return nil
	}

	// NOTE: As long as the token comes from the database we can trust it and do not have to validate it.
	// Each service that the request ultimately goes to will also validate before it uses the token
	var expiresAt int64 = -1
	if !accessToken.ExpiresAt.IsZero() {
		expiresAt = accessToken.ExpiresAt.Unix()
	}
	slog.Debug(
		"PROXY AUTH MIDDLEWARE",
		"message",
		"injected token in header",
		"header",
		headerKey,
		"token",
		accessToken.String(),
		"requestID",
		utils.GetRequestID(c),
	)
	c.Request().Header.Set(headerKey, accessToken.Value)
	if expiresAt != -1 {
		c.Request().Header.Set("Gitlab-Access-Token-Expires-At", strconv.FormatInt(expiresAt, 10))
	}
	return nil
}
