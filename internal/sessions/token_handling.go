package sessions

import (
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const tokenExpiryMargin time.Duration = time.Duration(3) * time.Minute

// GetAccessTokenFromContext retrieves an access token from the current context
func (sh *SessionHandler) GetAccessTokenFromContext(key string, c echo.Context) (models.AuthToken, error) {
	tokenRaw := c.Get(key)
	if tokenRaw != nil {
		token, ok := tokenRaw.(models.AuthToken)
		if !ok {
			return models.AuthToken{}, gwerrors.ErrTokenParse
		}
		if token.Expired() {
			return models.AuthToken{}, gwerrors.ErrTokenExpired
		}
		if token.Type != models.AccessTokenType {
			return models.AuthToken{}, gwerrors.ErrTokenType
		}
		return token, nil
	}
	return models.AuthToken{}, gwerrors.ErrTokenNotFound
}

func (sh *SessionHandler) GetAccessToken(c echo.Context, session Session, providerID string) (models.AuthToken, error) {
	if session.TokenIDs == nil {
		session.TokenIDs = models.SerializableMap{}
	}
	tokenID, tokenExists := session.TokenIDs[providerID]
	if !tokenExists {
		return models.AuthToken{}, gwerrors.ErrTokenNotFound
	}
	// check if the access token is already in the request context
	token, err := sh.GetAccessTokenFromContext(sh.accessTokenKey(tokenID), c)
	if err == nil {
		return token, nil
	}

	// token, err = sh.tokenStore.GetAccessToken(c.Request().Context(), tokenID)
	token, err = sh.tokenRefresher.GetFreshAccessToken(c.Request().Context(), tokenID)
	if err != nil {
		if err == redis.Nil {
			return models.AuthToken{}, gwerrors.ErrTokenNotFound
		} else {
			return models.AuthToken{}, err
		}
	}
	if token.Expired() {
		return models.AuthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (sh *SessionHandler) SaveTokens(c echo.Context, session *Session, tokens AuthTokenSet) error {
	err := tokens.ValidateTokensType()
	if err != nil {
		return err
	}
	// Update the session's token IDs
	providerID := tokens.AccessToken.ProviderID
	if session.TokenIDs == nil {
		session.TokenIDs = models.SerializableMap{}
	}
	session.TokenIDs[providerID] = tokens.AccessToken.ID
	// err = sh.tokenStore.SetAccessToken(c.Request().Context(), *session, tokens.AccessToken)
	expiresAt := sh.getTokenExpiration(tokens, *session)
	err = sh.tokenStore.SetAccessToken(c.Request().Context(), tokens.AccessToken)
	if err != nil {
		return err
	}
	err = sh.tokenStore.SetAccessTokenExpiry(c.Request().Context(), tokens.AccessToken, expiresAt)
	if err != nil {
		return err
	}
	// err = sh.tokenStore.SetRefreshToken(c.Request().Context(), *session, tokens.RefreshToken)
	err = sh.tokenStore.SetRefreshToken(c.Request().Context(), tokens.RefreshToken)
	if err != nil {
		return err
	}
	err = sh.tokenStore.SetRefreshTokenExpiry(c.Request().Context(), tokens.RefreshToken, expiresAt)
	if err != nil {
		return err
	}
	// err = sh.tokenStore.SetIDToken(c.Request().Context(), *session, tokens.IDToken)
	err = sh.tokenStore.SetIDToken(c.Request().Context(), tokens.IDToken)
	if err != nil {
		return err
	}
	err = sh.tokenStore.SetIDTokenExpiry(c.Request().Context(), tokens.RefreshToken, expiresAt)
	if err != nil {
		return err
	}
	return nil
}

func (SessionHandler) accessTokenKey(tokenID string) string {
	return AccessTokenCtxKey + ":" + tokenID
}

// getTokenExpiration returns the max session expiration unless the provider is GitLab, in which case there is no expiration
func (SessionHandler) getTokenExpiration(tokens AuthTokenSet, session Session) time.Time {
	providerID := tokens.AccessToken.ProviderID
	if providerID == "gitlab" {
		return time.Time{}
	}
	return session.CreatedAt.Add(session.MaxTTL())
}
