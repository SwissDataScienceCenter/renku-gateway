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
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
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
			"access token expires soon",
			"tokenID",
			tokenID,
			"providerID",
			token.ProviderID,
		)
		newTokenSet, err := ts.refreshAccessToken(ctx, token)
		if err != nil {
			slog.Info(
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

	if token.ExpiresSoon(ts.ExpiryMargin()) {
		slog.Debug(
			"TOKEN STORE",
			"message",
			"ID token expires soon",
			"tokenID",
			tokenID,
			"providerID",
			token.ProviderID,
		)
		newTokenSet, err := ts.refreshAccessToken(ctx, token)
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
			reloadedToken, err := ts.tokenRepo.GetIDToken(ctx, tokenID)
			if err != nil {
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
					"tokenID",
					tokenID,
					"providerID",
					token.ProviderID,
				)
			}
		}
	}
	if token.Expired() {
		return models.AuthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (ts *TokenStore) refreshAccessToken(ctx context.Context, token models.AuthToken) (sessions.AuthTokenSet, error) {
	refreshToken, err := ts.tokenRepo.GetRefreshToken(ctx, token.ID)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "GetRefreshToken failed", "error", err)
		return sessions.AuthTokenSet{}, err
	}
	freshTokens, err := ts.providerStore.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "RefreshAccessToken failed", "error", err)
		return sessions.AuthTokenSet{}, err
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
	err = ts.tokenRepo.SetAccessToken(ctx, freshTokens.AccessToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "SetAccessToken failed", "error", err)
		return sessions.AuthTokenSet{}, err
	}
	err = ts.tokenRepo.SetRefreshToken(ctx, freshTokens.RefreshToken)
	if err != nil {
		slog.Error("TOKEN STORE", "message", "SetRefreshToken failed", "error", err)
		return sessions.AuthTokenSet{}, err
	}
	if freshTokens.IDToken.ID != "" {
		err = ts.tokenRepo.SetIDToken(ctx, freshTokens.IDToken)
		if err != nil {
			slog.Error("TOKEN STORE", "message", "SetIDToken failed", "error", err)
			return sessions.AuthTokenSet{}, err
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

// func (ts *TokenStore) GetIDToken(ctx context.Context, tokenID string) (models.AuthToken, error) {
// 	return ts.tokenRepo.GetIDToken(ctx, tokenID)
// }

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
