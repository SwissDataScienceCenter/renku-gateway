package tokenrefresher2

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/redis/go-redis/v9"
)

type TokenRefresher struct {
	ExpiryMarginMinutes int

	providerStore oidc.ClientStore
	tokenStore    RefresherTokenStore
}

func (tr *TokenRefresher) GetFreshAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	token, err := tr.tokenStore.GetAccessToken(ctx, tokenID)
	if err != nil {
		if err == redis.Nil {
			return models.AuthToken{}, gwerrors.ErrTokenNotFound
		} else {
			return models.AuthToken{}, err
		}
	}

	if token.ExpiresSoon(tr.ExpiryMargin()) {
		slog.Debug(
			"TOKEN REFRESHER",
			"message",
			"token expires soon",
			"tokenID",
			tokenID,
			"providerID",
			token.ProviderID,
		)
		newAccessToken, err := tr.refreshAccessToken(ctx, token)
		if err != nil {
			slog.Debug(
				"TOKEN REFRESHER",
				"message",
				"refreshAccessToken failed, will try to reload the token",
				"tokenID",
				tokenID,
				"providerID",
				token.ProviderID,
			)
			reloadedToken, err := tr.tokenStore.GetAccessToken(ctx, tokenID)
			if err != nil {
				token = reloadedToken
			}
		} else {
			token = newAccessToken
		}
	}
	if token.Expired() {
		return models.AuthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (tr TokenRefresher) ExpiryMargin() time.Duration {
	return time.Duration(tr.ExpiryMarginMinutes) * time.Minute
}

func (tr *TokenRefresher) refreshAccessToken(ctx context.Context, token models.AuthToken) (models.AuthToken, error) {
	refreshToken, err := tr.tokenStore.GetRefreshToken(ctx, token.ID)
	if err != nil {
		slog.Error("TOKEN REFRESHER", "message", "GetRefreshToken failed", "error", err)
		return models.AuthToken{}, err
	}
	newAccessToken, newRefreshToken, err := tr.providerStore.RefreshAccessToken(refreshToken)
	if err != nil {
		slog.Error("TOKEN REFRESHER", "message", "RefreshAccessToken failed", "error", err)
		return models.AuthToken{}, err
	}
	// Update the access and refresh tokens in place
	newAccessToken.ID = token.ID
	newAccessToken.SessionID = token.SessionID
	newRefreshToken.ID = token.ID
	newRefreshToken.SessionID = token.SessionID
	err = tr.tokenStore.SetAccessToken(ctx, newAccessToken)
	if err != nil {
		slog.Error("TOKEN REFRESHER", "message", "SetAccessToken failed", "error", err)
		return models.AuthToken{}, err
	}
	err = tr.tokenStore.SetRefreshToken(ctx, newRefreshToken)
	if err != nil {
		slog.Error("TOKEN REFRESHER", "message", "SetRefreshToken failed", "error", err)
		return models.AuthToken{}, err
	}
	return newAccessToken, nil
}

type TokenRefresherOption func(*TokenRefresher) error

func WithExpiryMarginMinutes(expiresSoonMinutes int) TokenRefresherOption {
	return func(tr *TokenRefresher) error {
		tr.ExpiryMarginMinutes = expiresSoonMinutes
		return nil
	}
}

func WithConfig(loginConfig config.LoginConfig) TokenRefresherOption {
	return func(tr *TokenRefresher) error {
		providerStore, err := oidc.NewClientStore(loginConfig.Providers)
		if err != nil {
			return err
		}
		tr.providerStore = providerStore
		return nil
	}
}

func WithTokenStore(store RefresherTokenStore) TokenRefresherOption {
	return func(tr *TokenRefresher) error {
		tr.tokenStore = store
		return nil
	}
}

// NewTokenRefresher creates a new TokenRefresher that handles refreshing access tokens which are expiring soon.
func NewTokenRefresher(options ...TokenRefresherOption) (TokenRefresher, error) {
	tr := TokenRefresher{}
	for _, opt := range options {
		err := opt(&tr)
		if err != nil {
			return TokenRefresher{}, err
		}
	}
	if tr.ExpiryMarginMinutes <= 0 {
		return TokenRefresher{}, fmt.Errorf("invalid value for ExpiryMarginMinutes (%d)", tr.ExpiryMarginMinutes)
	}
	if tr.providerStore == nil {
		return TokenRefresher{}, fmt.Errorf("OIDC providers not initialized")
	}
	if tr.tokenStore == nil {
		return TokenRefresher{}, fmt.Errorf("token store not initialized")
	}
	return tr, nil
}
