package revproxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type AuthOption func(*Auth)

type TokenHandler func(c echo.Context, token models.OauthToken) error

func InjectInHeader(headerKey string) AuthOption {
	return func(a *Auth) {
		a.tokenHandler = func(c echo.Context, token models.OauthToken) error {
			slog.Info(
				"PROXY AUTH MIDDLEWARE",
				"message",
				"injected token in header",
				"header",
				headerKey,
				"providerID",
				a.providerID,
				"tokenID",
				token.ID,
				"tokenType",
				token.Type,
				"requestID",
				c.Request().Header.Get("X-Request-ID"),
			)
			c.Request().Header.Set(headerKey, token.Value)
			return nil
		}
	}
}

func InjectBearerToken() AuthOption {
	return func(a *Auth) {
		a.tokenHandler = func(c echo.Context, token models.OauthToken) error {
			slog.Info(
				"PROXY AUTH MIDDLEWARE",
				"message",
				"injected token in header",
				"header",
				echo.HeaderAuthorization,
				"providerID",
				a.providerID,
				"tokenID",
				token.ID,
				"tokenType",
				token.Type,
				"requestID",
				c.Request().Header.Get("X-Request-ID"),
			)
			c.Request().Header.Set(echo.HeaderAuthorization, fmt.Sprintf("Bearer %s", token.Value))
			return nil
		}
	}
}

func WithTokenHandler(handler TokenHandler) AuthOption {
	return func(a *Auth) {
		a.tokenHandler = handler
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

// Auth generates middleware that will inject tokens in the proxied http requests
type Auth struct {
	tokenHandler TokenHandler
	providerID   string
	tokenType    models.OauthTokenType
}

func NewAuth(options ...AuthOption) *Auth {
	auth := Auth{providerID: "renku", tokenType: models.AccessTokenType}
	for _, opt := range options {
		opt(&auth)
	}
	return &auth
}

func (a *Auth) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sessionRaw := c.Get(models.SessionCtxKey)
			if sessionRaw == nil {
				return gwerrors.ErrSessionNotFound
			}
			session, ok := sessionRaw.(models.Session)
			if !ok {
				return gwerrors.ErrSessionParse
			}
			var token models.OauthToken
			var err error
			if a.tokenType == models.AccessTokenType {
				token, err = session.GetAccessToken(c.Request().Context(), a.providerID)
			} else if a.tokenType == models.IDTokenType {
				token, err = session.GetIDToken(c.Request().Context(), a.providerID)
			} else if a.tokenType == models.RefreshTokenType {
				token, err = session.GetRefreshToken(c.Request().Context(), a.providerID)
			} else {
				return fmt.Errorf("unknown token type in authentication middleware %s", a.tokenType)
			}
			if err != nil {
				switch err {
				case gwerrors.ErrTokenNotFound:
					slog.Info(
						"PROXY AUTH MIDDLEWARE",
						"message",
						"token not found continuing with middleware chain",
						"sessionID",
						session.ID,
						"providerID",
						a.providerID,
						"tokenType",
						a.tokenType,
						"requestID",
						c.Request().Header.Get("X-Request-ID"),
					)
					return next(c)
				case gwerrors.ErrTokenExpired:
					slog.Info(
						"PROXY AUTH MIDDLEWARE",
						"message",
						"token expired continuing with middleware chain",
						"sessionID",
						session.ID,
						"providerID",
						a.providerID,
						"tokenType",
						a.tokenType,
						"requestID",
						c.Request().Header.Get("X-Request-ID"),
					)
					return next(c)
				default:
					return err
				}
			}
			if a.tokenHandler == nil {
				return fmt.Errorf("missing token handler for the authenitcation middelware")
			}
			err = a.tokenHandler(c, token)
			if err != nil {
				return err
			}
			return next(c)
		}
	}
}

var notebooksGitlabAccessTokenHandler TokenHandler = func(c echo.Context, accessToken models.OauthToken) error {
	// NOTE: As long as the token comes from the database we can trust it and do nto have to validate it.
	// Each service that the request ultimately goes to will also validate before it users the token
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
	c.Request().Header.Set("Renku-Auth-Git-Credentials", headerVal)
	return nil
}

var coreSvcRenkuAccessTokenHandler TokenHandler = func(c echo.Context, accessToken models.OauthToken) error {
	extractClaim := func(claims jwt.MapClaims, key string) (string, error) {
		valRaw, found := claims["email"]
		if !found {
			return "", fmt.Errorf("cannot find %s claim in access token for core service", key)
		}
		val, ok := valRaw.(string)
		if !ok {
			return "", fmt.Errorf("cannot parse %s claim as string in access token for core service", key)
		}
		return val, nil
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedJWT, _, err := parser.ParseUnverified(accessToken.Value, jwt.MapClaims{})
	if err != nil {
		return err
	}
	claims, ok := parsedJWT.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("cannot parse claims")
	}
	email, err := extractClaim(claims, "email")
	if err != nil {
		return err
	}
	sub, err := extractClaim(claims, "sub")
	if err != nil {
		return err
	}
	name, err := extractClaim(claims, "name")
	if err != nil {
		return err
	}
	c.Request().Header.Set("Renku-user-id", sub)
	c.Request().Header.Set("Renku-user-email", email)
	c.Request().Header.Set("Renku-user-fullname", name)
	c.Request().Header.Set(echo.HeaderAuthorization, fmt.Sprintf("Bearer %s", accessToken.Value))
	return nil
}

// Sets up Basic Auth for Gitlab 
var gitlabCliTokenHandler TokenHandler = func(c echo.Context, accessToken models.OauthToken) error {
	if accessToken.Value == "" {
		return nil 
	}
	c.Request().SetBasicAuth("oauth2", accessToken.Value)
	return nil
}
