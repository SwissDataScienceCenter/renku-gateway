package revproxy

import (
	"io"
	"net/http"
	"net/url"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

// Retrieves tokens from "gateway-auth"
func legacyAuth(authURL *url.URL, keycloakURL *url.URL) echo.MiddlewareFunc {
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
			// Send token request to "gateway-auth"
			req, err := http.NewRequestWithContext(
				c.Request().Context(),
				"GET",
				authURL.String(),
				nil,
			)
			if err != nil {
				return err
			}
			req.Header = c.Request().Header.Clone()
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			// The authentication request was rejected, return the authentication service response and status code
			if res.StatusCode >= 300 || res.StatusCode < 200 {
				defer res.Body.Close()
				for name, values := range res.Header {
					c.Response().Header()[name] = values
				}
				c.Response().WriteHeader(res.StatusCode)
				_, err = io.Copy(c.Response().Writer, res.Body)
				return err
			}
			// The authentication request was successful, save tokens to session
			renkuAccessTokenRaw := res.Header.Get("Renku-Auth-Access-Token")
			var renkuAccessToken models.OauthToken
			claims := new(oidc.TokenClaims)
			_, err = oidc.ParseToken(renkuAccessTokenRaw, claims)
			if err != nil {
				renkuAccessToken = models.OauthToken{}
			} else {
				renkuAccessToken = models.OauthToken{
					ID:         claims.ClientID,
					Value:      renkuAccessTokenRaw,
					ExpiresAt:  claims.GetExpiration(),
					TokenURL:   keycloakURL.String(),
					ProviderID: "renku",
					Type:       models.AccessTokenType,
				}
			}
			renkuRefreshTokenRaw := res.Header.Get("Renku-Auth-Refresh-Token")
			var renkuRefreshToken models.OauthToken
			claims = new(oidc.TokenClaims)
			_, err = oidc.ParseToken(renkuRefreshTokenRaw, claims)
			if err != nil {
				renkuRefreshToken = models.OauthToken{}
			} else {
				renkuRefreshToken = models.OauthToken{
					ID:         claims.ClientID,
					Value:      renkuRefreshTokenRaw,
					ExpiresAt:  claims.GetExpiration(),
					TokenURL:   keycloakURL.String(),
					ProviderID: "renku",
					Type:       models.RefreshTokenType,
				}
			}
			renkuIdTokenRaw := res.Header.Get("Renku-Auth-Id-Token")
			var renkuIdToken models.OauthToken
			claims = new(oidc.TokenClaims)
			_, err = oidc.ParseToken(renkuIdTokenRaw, claims)
			if err != nil {
				renkuIdToken = models.OauthToken{}
			} else {
				renkuIdToken = models.OauthToken{
					ID:         claims.ClientID,
					Value:      renkuIdTokenRaw,
					ExpiresAt:  claims.GetExpiration(),
					TokenURL:   keycloakURL.String(),
					ProviderID: "renku",
					Type:       models.IDTokenType,
				}
			}

			// accessToken := models.OauthToken{
			// 	ID:         id,
			// 	Type:       models.AccessTokenType,
			// 	Value:      tokens.AccessToken,
			// 	TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			// 	ExpiresAt:  tokens.Expiry,
			// 	ProviderID: c.getID(),
			// }
			// refreshToken := models.OauthToken{
			// 	ID:         id,
			// 	Type:       models.RefreshTokenType,
			// 	Value:      tokens.RefreshToken,
			// 	TokenURL:   client.OAuthConfig().Endpoint.TokenURL,
			// 	ProviderID: c.getID(),
			// }
			// idToken := models.OauthToken{
			// 	ID:         id,
			// 	Type:       models.IDTokenType,
			// 	Value:      tokens.IDToken,
			// 	ExpiresAt:  tokens.IDTokenClaims.GetExpiration(),
			// 	ProviderID: c.getID(),
			// }

			// headers["Authorization"] = f"Bearer {access_token}"
			// headers["Renku-Auth-Access-Token"] = access_token
			// headers["Renku-Auth-Refresh-Token"] = keycloak_oidc_client.refresh_token
			// headers["Renku-Auth-Id-Token"] = keycloak_oidc_client.token["id_token"]
			// headers["Gitlab-Access-Token"] = gitlab_oauth_client.access_token
			session.SaveTokens(c.Request().Context(), renkuAccessToken, renkuRefreshToken, renkuIdToken, "")

			return next(c)
		}
	}
}
