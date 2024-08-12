package sessions

import (
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

// GetAccessTokenFromContext retrieves an access token from the current context
func (sessions *SessionStore) GetAccessTokenFromContext(key string, c echo.Context) (models.AuthToken, error) {
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

func (sessions *SessionStore) GetAccessToken(c echo.Context, session models.Session, providerID string) (models.AuthToken, error) {
	if session.TokenIDs == nil {
		session.TokenIDs = models.SerializableMap{}
	}
	tokenID, tokenExists := session.TokenIDs[providerID]
	if !tokenExists {
		return models.AuthToken{}, gwerrors.ErrTokenNotFound
	}
	// check if the access token is already in the request context
	token, err := sessions.GetAccessTokenFromContext(sessions.accessTokenKey(tokenID), c)
	if err == nil {
		return token, nil
	}

	token, err = sessions.tokenStore.GetFreshAccessToken(c.Request().Context(), tokenID)
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

func (sessions *SessionStore) SaveTokens(c echo.Context, session *models.Session, tokens AuthTokenSet) error {
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
	expiresAt := sessions.getTokenExpiration(tokens, *session)
	err = sessions.tokenStore.SetAccessToken(c.Request().Context(), tokens.AccessToken)
	if err != nil {
		return err
	}
	err = sessions.tokenStore.SetAccessTokenExpiry(c.Request().Context(), tokens.AccessToken, expiresAt)
	if err != nil {
		return err
	}
	err = sessions.tokenStore.SetRefreshToken(c.Request().Context(), tokens.RefreshToken)
	if err != nil {
		return err
	}
	err = sessions.tokenStore.SetRefreshTokenExpiry(c.Request().Context(), tokens.RefreshToken, expiresAt)
	if err != nil {
		return err
	}
	err = sessions.tokenStore.SetIDToken(c.Request().Context(), tokens.IDToken)
	if err != nil {
		return err
	}
	err = sessions.tokenStore.SetIDTokenExpiry(c.Request().Context(), tokens.IDToken, expiresAt)
	if err != nil {
		return err
	}
	return nil
}

func (SessionStore) accessTokenKey(tokenID string) string {
	return AccessTokenCtxKey + ":" + tokenID
}

// getTokenExpiration returns the max session expiration unless the provider is GitLab, in which case there is no expiration
func (SessionStore) getTokenExpiration(tokens AuthTokenSet, session models.Session) time.Time {
	providerID := tokens.AccessToken.ProviderID
	if providerID == "gitlab" {
		return time.Time{}
	}
	return session.CreatedAt.Add(session.MaxTTL())
}
