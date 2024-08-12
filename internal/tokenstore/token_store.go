package tokenstore

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

type TokenStore struct {
	ExpiryMarginMinutes int

	providerStore oidc.ClientStore
	tokenRepo     models.TokenRepository
}

func (ts TokenStore) ExpiryMargin() time.Duration {
	return time.Duration(ts.ExpiryMarginMinutes) * time.Minute
}

func (ts *TokenStore) GetFreshAccessToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	token, err := ts.tokenRepo.GetAccessToken(ctx, tokenID)
	if err != nil {
		if err == redis.Nil {
			return models.AuthToken{}, gwerrors.ErrTokenNotFound
		} else {
			return models.AuthToken{}, err
		}
	}

	if token.ExpiresSoon(ts.ExpiryMargin()) {
		slog.Debug(
			"TOKEN STORE",
			"message",
			"token expires soon",
			"tokenID",
			tokenID,
			"providerID",
			token.ProviderID,
		)
		newAccessToken, err := ts.refreshAccessToken(ctx, token)
		if err != nil {
			slog.Debug(
				"TOKEN STORE",
				"message",
				"refreshAccessToken failed, will try to reload the token",
				"tokenID",
				tokenID,
				"providerID",
				token.ProviderID,
			)
			reloadedToken, err := ts.tokenRepo.GetAccessToken(ctx, tokenID)
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

func (ts *TokenStore) refreshAccessToken(ctx context.Context, token models.AuthToken) (models.AuthToken, error) {
	refreshToken, err := ts.tokenRepo.GetRefreshToken(ctx, token.ID)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "GetRefreshToken failed", "error", err)
		return models.AuthToken{}, err
	}
	newAccessToken, newRefreshToken, err := ts.providerStore.RefreshAccessToken(refreshToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "RefreshAccessToken failed", "error", err)
		return models.AuthToken{}, err
	}
	// Update the access and refresh tokens in place
	newAccessToken.ID = token.ID
	newAccessToken.SessionID = token.SessionID
	newRefreshToken.ID = token.ID
	newRefreshToken.SessionID = token.SessionID
	err = ts.tokenRepo.SetAccessToken(ctx, newAccessToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "SetAccessToken failed", "error", err)
		return models.AuthToken{}, err
	}
	err = ts.tokenRepo.SetRefreshToken(ctx, newRefreshToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "SetRefreshToken failed", "error", err)
		return models.AuthToken{}, err
	}
	return newAccessToken, nil
}

func (ts *TokenStore) SetAccessToken(ctx context.Context, token models.AuthToken) error {
	return ts.tokenRepo.SetAccessToken(ctx, token)
}

func (ts *TokenStore) SetAccessTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	return ts.tokenRepo.SetAccessTokenExpiry(ctx, token, expiresAt)
}

func (ts *TokenStore) SetRefreshToken(ctx context.Context, token models.AuthToken) error {
	return ts.tokenRepo.SetRefreshToken(ctx, token)
}

func (ts *TokenStore) SetRefreshTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	return ts.tokenRepo.SetRefreshTokenExpiry(ctx, token, expiresAt)
}
func (ts *TokenStore) SetIDToken(ctx context.Context, token models.AuthToken) error {
	return ts.tokenRepo.SetIDToken(ctx, token)
}

func (ts *TokenStore) SetIDTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	return ts.tokenRepo.SetIDTokenExpiry(ctx, token, expiresAt)
}

type TokenRefresherOption func(*TokenStore) error

func WithExpiryMarginMinutes(expiresSoonMinutes int) TokenRefresherOption {
	return func(ts *TokenStore) error {
		ts.ExpiryMarginMinutes = expiresSoonMinutes
		return nil
	}
}

func WithConfig(loginConfig config.LoginConfig) TokenRefresherOption {
	return func(ts *TokenStore) error {
		providerStore, err := oidc.NewClientStore(loginConfig.Providers)
		if err != nil {
			return err
		}
		ts.providerStore = providerStore
		return nil
	}
}

func WithTokenRepository(tokenRepo models.TokenRepository) TokenRefresherOption {
	return func(ts *TokenStore) error {
		ts.tokenRepo = tokenRepo
		return nil
	}
}

// NewTokenStore creates a new TokenRefresher that handles refreshing access tokens which are expiring soon.
func NewTokenStore(options ...TokenRefresherOption) (*TokenStore, error) {
	ts := TokenStore{}
	for _, opt := range options {
		err := opt(&ts)
		if err != nil {
			return &TokenStore{}, err
		}
	}
	if ts.ExpiryMarginMinutes <= 0 {
		return &TokenStore{}, fmt.Errorf("invalid value for ExpiryMarginMinutes (%d)", ts.ExpiryMarginMinutes)
	}
	if ts.providerStore == nil {
		return &TokenStore{}, fmt.Errorf("OIDC providers not initialized")
	}
	if ts.tokenRepo == nil {
		return &TokenStore{}, fmt.Errorf("token repository not initialized")
	}
	return &ts, nil
}
