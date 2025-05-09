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
	ExpiryMargin time.Duration

	providerStore oidc.ClientStore
	tokenRepo     models.TokenRepository
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

	if token.ExpiresSoon(ts.ExpiryMargin) {
		slog.Debug(
			"TOKEN STORE",
			"message",
			"access token expires soon",
			"token",
			token.String(),
		)
		newTokenSet, err := ts.refreshAccessToken(ctx, token)
		if err != nil {
			slog.Info(
				"TOKEN STORE",
				"message",
				"refreshAccessToken failed, will try to reload the token",
				"token",
				token.String(),
			)
			// Attempt to reload the token, it may have been refreshed by another instance.
			reloadedToken, err := ts.tokenRepo.GetAccessToken(ctx, tokenID)
			if err == nil {
				token = reloadedToken
			}
		} else {
			token = newTokenSet.AccessToken
		}
	}
	if token.Expired() {
		return models.AuthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (ts *TokenStore) GetFreshIDToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	token, err := ts.tokenRepo.GetIDToken(ctx, tokenID)
	if err != nil {
		if err == redis.Nil {
			return models.AuthToken{}, gwerrors.ErrTokenNotFound
		} else {
			return models.AuthToken{}, err
		}
	}

	if token.ExpiresSoon(ts.ExpiryMargin) {
		slog.Debug(
			"TOKEN STORE",
			"message",
			"ID token expires soon",
			"token",
			token.String(),
		)
		newTokenSet, err := ts.refreshAccessToken(ctx, token)
		if err != nil {
			slog.Info(
				"TOKEN STORE",
				"message",
				"refreshAccessToken failed, will try to reload the token",
				"token",
				token.String(),
			)
			// Attempt to reload the token, it may have been refreshed by another instance.
			reloadedToken, err := ts.tokenRepo.GetIDToken(ctx, tokenID)
			if err == nil {
				token = reloadedToken
			}
		} else {
			if newTokenSet.IDToken.ID != "" {
				token = newTokenSet.IDToken
			} else {
				slog.Error(
					"TOKEN STORE",
					"message",
					"refreshAccessToken did not provide a new ID token",
					"token",
					token.String(),
				)
			}
		}
	}
	if token.Expired() {
		return models.AuthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (ts *TokenStore) refreshAccessToken(ctx context.Context, token models.AuthToken) (models.AuthTokenSet, error) {
	refreshToken, err := ts.tokenRepo.GetRefreshToken(ctx, token.ID)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "GetRefreshToken failed", "error", err)
		return models.AuthTokenSet{}, err
	}
	// We want to perform this whole operation without cancelling
	childCtx := context.WithoutCancel(ctx)
	freshTokens, err := ts.providerStore.RefreshAccessToken(childCtx, refreshToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "RefreshAccessToken failed", "error", err)
		return models.AuthTokenSet{}, err
	}
	// Update the access, refresh and ID tokens in place
	freshTokens.AccessToken.ID = token.ID
	freshTokens.AccessToken.SessionID = token.SessionID
	freshTokens.RefreshToken.ID = token.ID
	freshTokens.RefreshToken.SessionID = token.SessionID
	if freshTokens.IDToken.ID != "" {
		freshTokens.IDToken.ID = token.ID
		freshTokens.IDToken.SessionID = token.SessionID
	}
	err = ts.tokenRepo.SetAccessToken(childCtx, freshTokens.AccessToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "SetAccessToken failed", "error", err)
		return models.AuthTokenSet{}, err
	}
	err = ts.tokenRepo.SetRefreshToken(childCtx, freshTokens.RefreshToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "SetRefreshToken failed", "error", err)
		return models.AuthTokenSet{}, err
	}
	if freshTokens.IDToken.ID != "" {
		err = ts.tokenRepo.SetIDToken(childCtx, freshTokens.IDToken)
		if err != nil {
			slog.Error("TOKEN STORE", "message", "SetIDToken failed", "error", err)
			return models.AuthTokenSet{}, err
		}
	}
	return freshTokens, nil
}

func (ts *TokenStore) SetAccessToken(ctx context.Context, token models.AuthToken) error {
	return ts.tokenRepo.SetAccessToken(ctx, token)
}

func (ts *TokenStore) SetAccessTokenExpiry(ctx context.Context, token models.AuthToken, expiresAt time.Time) error {
	return ts.tokenRepo.SetAccessTokenExpiry(ctx, token, expiresAt)
}

func (ts *TokenStore) GetRefreshToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
	return ts.tokenRepo.GetRefreshToken(ctx, tokenID)
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

func WithExpiryMargin(expiresSoon time.Duration) TokenRefresherOption {
	return func(ts *TokenStore) error {
		ts.ExpiryMargin = expiresSoon
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
	if ts.ExpiryMargin <= time.Duration(0) {
		return &TokenStore{}, fmt.Errorf("invalid value for ExpiryMargin (%d)", ts.ExpiryMargin)
	}
	if ts.providerStore == nil {
		return &TokenStore{}, fmt.Errorf("OIDC providers not initialized")
	}
	if ts.tokenRepo == nil {
		return &TokenStore{}, fmt.Errorf("token repository not initialized")
	}
	return &ts, nil
}
