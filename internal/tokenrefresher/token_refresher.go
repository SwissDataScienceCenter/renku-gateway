package tokenrefresher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/SwissDataScienceCenter/renku-gateway/internal/oidc"
	"github.com/go-co-op/gocron"
)

type TokenRefresher struct {
	ExpiresSoonMinutes int

	providerStore oidc.ClientStore
	sessionStore  RefresherSessionStore
	tokenStore    RefresherTokenStore
}

func (tr *TokenRefresher) GetScheduler() (*gocron.Scheduler, error) {
	s := gocron.NewScheduler(time.UTC)

	refreshExpiringTokensTask := func(job gocron.Job) {
		err := tr.refreshExpiringTokens(job.Context())
		if err != nil {
			slog.Error("TOKEN REFRESHER", "message", "refreshExpiringTokens failed", "error", err)
		}
	}

	_, err := s.Every(1).
		Minutes().
		DoWithJobDetails(refreshExpiringTokensTask)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (tr *TokenRefresher) refreshExpiringTokens(ctx context.Context) error {
	// Get a list of expiring access tokens ids in the next minsToExpiration minutes
	expiryEnd := time.Now().Add(time.Duration(tr.ExpiresSoonMinutes) * time.Minute)
	expiringTokenIDs, err := tr.tokenStore.GetExpiringAccessTokenIDs(ctx, expiryEnd)
	if err != nil {
		slog.Error("TOKEN REFRESHER", "message", "GetExpiringAccessTokenIDs failed", "error", err)
		return err
	}
	errorTokenIDs := []string{}
	for _, tokenID := range expiringTokenIDs {
		// Get the refresh token
		refreshToken, err := tr.tokenStore.GetRefreshToken(ctx, tokenID)
		if err != nil {
			slog.Error("TOKEN REFRESHER", "message", "GetRefreshToken failed", "error", err)
			errorTokenIDs = append(errorTokenIDs, tokenID)
			continue
		}
		// Verify if the session is still active
		session, err := tr.sessionStore.GetSession(ctx, refreshToken.SessionID)
		if err != nil {
			slog.Error("TOKEN REFRESHER", "message", "GetSession failed", "error", err)
			errorTokenIDs = append(errorTokenIDs, tokenID)
			continue
		}
		if session.Expired() {
			slog.Error("TOKEN REFRESHER", "message", "Session has expired", "tokenID", tokenID, "sessionID", session.ID)
			errorTokenIDs = append(errorTokenIDs, tokenID)
			continue
		}
		// Call the refresh endpoint
		// newAccessToken, newRefreshToken, err := tr.providerStore.RefreshAccessToken(refreshToken)
		_, _, err = tr.providerStore.RefreshAccessToken(refreshToken)
		if err != nil {
			slog.Error("TOKEN REFRESHER", "message", "RefreshAccessToken failed", "error", err)
			errorTokenIDs = append(errorTokenIDs, tokenID)
			continue
		}
		// Set the refreshed access and refresh token values into the token store
		// err = tr.tokenStore.SetAccessToken(ctx, session, newAccessToken)
		if err != nil {
			slog.Error("TOKEN REFRESHER", "message", "SetAccessToken failed", "error", err)
			errorTokenIDs = append(errorTokenIDs, tokenID)
			continue
		}
		// err = tr.tokenStore.SetRefreshToken(ctx, session, newRefreshToken)
		if err != nil {
			slog.Error("TOKEN REFRESHER", "message", "SetRefreshToken failed", "error", err)
			errorTokenIDs = append(errorTokenIDs, tokenID)
			continue
		}
	}

	slog.Info(
		"TOKEN REFRESHER", "message",
		fmt.Sprintf(
			"%v/%v expiring access tokens refreshed",
			len(expiringTokenIDs)-len(errorTokenIDs),
			len(expiringTokenIDs),
		),
	)

	if len(errorTokenIDs) != 0 {
		return fmt.Errorf("some token IDs could not be refreshed %v", errorTokenIDs)
	}
	return nil
}

type TokenRefresherOption func(*TokenRefresher) error

func WithExpiresSoonMinutes(expiresSoonMinutes int) TokenRefresherOption {
	return func(tr *TokenRefresher) error {
		tr.ExpiresSoonMinutes = expiresSoonMinutes
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

func WithSessionStore(store RefresherSessionStore) TokenRefresherOption {
	return func(tr *TokenRefresher) error {
		tr.sessionStore = store
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
	if tr.ExpiresSoonMinutes <= 0 {
		return TokenRefresher{}, fmt.Errorf("invalid value for ExpiresSoonMinutes (%d)", tr.ExpiresSoonMinutes)
	}
	if tr.providerStore == nil {
		return TokenRefresher{}, fmt.Errorf("OIDC providers not initialized")
	}
	if tr.sessionStore == nil {
		return TokenRefresher{}, fmt.Errorf("session store not initialized")
	}
	if tr.tokenStore == nil {
		return TokenRefresher{}, fmt.Errorf("token store not initialized")
	}
	return tr, nil
}
