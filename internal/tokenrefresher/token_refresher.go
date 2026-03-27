package tokenrefresher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// TokenRefresher handles keeping refresh tokens alive in the background
type TokenRefresher struct {
	tokenRefreshRepository models.TokenRefreshRepository
	// ticker for periodically refreshing tokens
	ticker *time.Ticker
	// stop channel to stop listening the ticker
	stop chan<- bool
}

func (tr *TokenRefresher) Start() {
	if tr.ticker != nil {
		return
	}
	stop := make(chan bool, 1)
	go tr.periodicTokensRefresh(stop)
	tr.ticker = time.NewTicker(time.Minute)
	tr.stop = stop
}

func (tr *TokenRefresher) Stop() {
	if tr.ticker == nil {
		return
	}
	tr.ticker.Stop()
	tr.stop <- true
	tr.ticker = nil
	tr.stop = nil
}

func (tr *TokenRefresher) periodicTokensRefresh(stop <-chan bool) {
	for {
		select {
		case <-stop:
			return
		case <-tr.ticker.C:
			slog.Info("TOKEN REFRESHER", "message", "tick")
			ctx := context.Background()
			getCtx, cancelGetCtx := context.WithTimeout(ctx, 10*time.Second)
			defer cancelGetCtx()
			now := time.Now()
			tokenIDs, err := tr.tokenRefreshRepository.GetExpiringRefreshTokenIDs(getCtx, now, now.Add(time.Hour))
			if err != nil {
				slog.Error("TOKEN REFRESHER", "message", "error getting expiring refresh tokens", "error", err)
				continue
			}
			slog.Info("TOKEN REFRESHER", "tokenIDs", tokenIDs)
		}
	}
}

type TokenRefresherOption func(*TokenRefresher) error

func WithTokenRefreshRepository(tokenRefreshRepository models.TokenRefreshRepository) TokenRefresherOption {
	return func(tr *TokenRefresher) error {
		tr.tokenRefreshRepository = tokenRefreshRepository
		return nil
	}
}

func NewTokenRefresher(options ...TokenRefresherOption) (*TokenRefresher, error) {
	tokenRefresher := TokenRefresher{}
	for _, opt := range options {
		err := opt(&tokenRefresher)
		if err != nil {
			return &TokenRefresher{}, err
		}
	}
	if tokenRefresher.tokenRefreshRepository == nil {
		return &TokenRefresher{}, fmt.Errorf("token refresh repository is not initialized")
	}
	return &tokenRefresher, nil
}
