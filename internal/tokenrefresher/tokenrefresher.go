// Package tokenrefresher refreshes oauth tokens stored by the gateway.
package tokenrefresher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/go-co-op/gocron"
)

// tokenReponse struct required to unmarshal the response from a POST token refresh request
type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	Type                  string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_expires_in"`
	Scope                 string `json:"scope"`
	CreatedAt             int64  `json:"created_at"`
}

func (t tokenResponse) String() string {
	return fmt.Sprintf(
		"CreatedAt: %v, Type: %v, ExpiresIn: %v, RefreshTokenExpiresIn: %v",
		t.CreatedAt,
		t.Type,
		t.ExpiresIn,
		t.RefreshTokenExpiresIn,
	)
}

// RefresherTokenStore is an interface used for refreshing tokens stored by the gateway
type RefresherTokenStore interface {
	GetRefreshToken(context.Context, string) (models.OauthToken, error)
	GetAccessToken(context.Context, string) (models.OauthToken, error)
	SetRefreshToken(context.Context, models.OauthToken) error
	SetAccessToken(context.Context, models.OauthToken) error
	GetExpiringAccessTokenIDs(context.Context, time.Time, time.Time) ([]string, error)
}

// ScheduleRefreshExpiringTokens intialises a gocron job to run refreshExpiringTokens at a specified interval
func ScheduleRefreshExpiringTokens(
	ctx context.Context,
	tokenStore RefresherTokenStore,
	gitlabClientID string,
	gitlabClientSecret string,
	minsToExpiration int,
) error {
	s := gocron.NewScheduler(time.UTC)
	job, err := s.Every(minsToExpiration).
		Minutes().
		Do(refreshExpiringTokens, ctx, tokenStore, gitlabClientID, gitlabClientSecret, minsToExpiration)
	s.StartBlocking()
	if err != nil {
		slog.Error("Starting gocron job failed", "error", err)
	} else {
		slog.Info("Job starting", "job", job)
	}
	return err
}

// refreshExpiringTokens refreshes tokens in the token store expiring in the next minsToExpiration minutes
func refreshExpiringTokens(
	ctx context.Context,
	tokenStore RefresherTokenStore,
	clientID string,
	clientSecret string,
	minsToExpiration int,
) error {
	// Get a list of expiring access tokens ids in the next minsToExpiration minutes
	expiringTokenIDs, err := tokenStore.GetExpiringAccessTokenIDs(
		ctx,
		time.Now(),
		time.Now().Add(time.Minute*time.Duration(minsToExpiration)),
	)
	if err != nil {
		slog.Error("GetExpiringAccessTokenIDs failed", "error", err)
		return err
	}

	// For each token id expiring in the next minsToExpiration minutes
	for _, expiringTokenID := range expiringTokenIDs {

		// Get the refresh and access tokens associated with the token ID
		myRefreshToken, err := tokenStore.GetRefreshToken(ctx, expiringTokenID)
		if err != nil {
			slog.Error("GetRefreshToken failed", "error", err)
			return err
		}

		myAccessToken, err := tokenStore.GetAccessToken(ctx, expiringTokenID)
		if err != nil {
			slog.Error("GetAccessToken failed", "error", err)
			return err
		}

		// Set the parameters required to refresh the tokens
		params := url.Values{}
		params.Add("client_id", clientID)
		params.Add("client_secret", clientSecret)
		params.Add("refresh_token", myRefreshToken.Value)
		params.Add("grant_type", "refresh_token")

		// Send the POST request to refresh the tokens
		resp, err := http.PostForm(myAccessToken.TokenURL, params)
		if err != nil {
			slog.Error("Request Failed", "error", err)
			return err
		}
		defer resp.Body.Close()

		// Decode JSON returned from the POST refresh request into a tokenResponse
		token := tokenResponse{}
		err = json.NewDecoder(resp.Body).Decode(&token)
		if err != nil {
			slog.Error("Decoding body failed", "error", err)
			return err
		}

		slog.Info("New token received")

		// Calculate the UNIX timestamp at which the newly refreshed access and refresh tokens will expire
		accessTokenExpiration := time.Unix(token.CreatedAt+token.ExpiresIn, 0)
		// Keycloak does not provide a created_at parameter.
		// Therefore, if the value of token.CreatedAt is 0,
		// we replace token.CreatedAt with time.Now()
		if token.CreatedAt == 0 {
			accessTokenExpiration = time.Now().Add(time.Second * time.Duration(token.ExpiresIn))
		}

		refreshTokenExpiration := time.Now().Add(time.Second * time.Duration(token.RefreshTokenExpiresIn))
		// Gitlab refresh tokens do not expire
		// (see https://gitlab.com/gitlab-org/gitlab/-/issues/340848#note_953496566).
		// Therefore, in the case that there is no refresh token expiration time,
		// we set a refresh token expiration time of 0.
		if token.RefreshTokenExpiresIn == 0 {
			refreshTokenExpiration = time.Unix(0, 0)
		}

		// Set the refreshed access and refresh token values into the token store
		err = tokenStore.SetAccessToken(ctx, models.OauthToken{
			ID:        myAccessToken.ID,
			Value:     token.AccessToken,
			ExpiresAt: accessTokenExpiration,
			TokenURL:  myAccessToken.TokenURL,
			Type:      myAccessToken.Type,
		})

		err = tokenStore.SetRefreshToken(ctx, models.OauthToken{
			ID:        myRefreshToken.ID,
			Value:     token.RefreshToken,
			ExpiresAt: refreshTokenExpiration,
		})
	}

	slog.Info(
		fmt.Sprintf(
			"%v expiring access tokens refreshed, evaluating again in %v minutes",
			len(expiringTokenIDs),
			minsToExpiration,
		),
	)
	return err
}
