package tokenrefresher

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
// )

// var ctx = context.Background()

// type DummyAdapter struct {
// 	err          error
// 	accessToken  models.AuthToken
// 	refreshToken models.AuthToken
// 	tokenID      string
// }

// func (d *DummyAdapter) GetRefreshToken(context.Context, string) (models.AuthToken, error) {
// 	return d.refreshToken, d.err
// }
// func (d *DummyAdapter) GetAccessToken(context.Context, string) (models.AuthToken, error) {
// 	return d.accessToken, d.err
// }
// func (d *DummyAdapter) SetRefreshToken(ctx context.Context, aRefreshToken models.AuthToken) error {
// 	d.refreshToken = aRefreshToken
// 	return d.err
// }
// func (d *DummyAdapter) SetAccessToken(ctx context.Context, anAccessToken models.AuthToken) error {
// 	d.accessToken = anAccessToken
// 	return d.err
// }
// func (d *DummyAdapter) GetExpiringAccessTokenIDs(context.Context, time.Time, time.Time) ([]string, error) {
// 	return []string{d.tokenID}, d.err
// }

// func TestRefreshExpiringTokensGitlab(t *testing.T) {

// 	log.Printf("Testing GitLab access token refresh")

// 	// Set dummy values for the 'existing' access and refresh tokens, and the oauth client id and secret
// 	tokenID := "rNDSNs005xrNvrgKZ5vJGCDqwA3VQ1MB"
// 	refreshTokenValue := "QG2RX43C81P5SNS1GACEMNKVT3SDBS"
// 	accessTokenValue := "C1SB4BC3HTP841TGVS4R4G5JEAVQT4W"
// 	clientID := "iPG5UPqrV6LiXiziLbj0CBGbDvWdPWwG"
// 	clientSecret := "9p9KBXSUj037qkR55mdS0yAAecBxbb8Q"

// 	// Set the dummy values we want the access and refresh tokens to have after refreshing them.
// 	refreshedAccessTokenValue := "6XGQJCST3BY1BZ7X5X78X2MLF0W1AUB5"
// 	refreshedRefreshTokenValue := "5EU358RBY51B88OP0JJ5S15WPSTCSCX3"
// 	refreshedTokenCreationTime := time.Now().Unix()

// 	// Set up test HTTP server that refresh requests will be sent to
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method == "POST" {
// 			log.Println(w, "Received POST request!")
// 			err := r.ParseForm()
// 			if err != nil {
// 				http.Error(w, err.Error(), http.StatusBadRequest)
// 				t.Fatal(err)
// 			}

// 			// Ensure the expected values are received by the test HTTP server
// 			if refreshTokenValue == r.PostForm["refresh_token"][0] {
// 				log.Printf("The refresh token posted is the correct value, %v\n", r.PostForm["refresh_token"][0])
// 			} else {
// 				t.Errorf("The refresh token posted is NOT the correct value, got %v want %v\n", r.PostForm["refresh_token"][0], refreshTokenValue)
// 			}

// 			if clientID == r.PostForm["client_id"][0] {
// 				log.Printf("The client ID posted is the correct value, %v\n", r.PostForm["client_id"][0])
// 			} else {
// 				t.Errorf("The client ID posted is NOT the correct value, got %v want %v\n", r.PostForm["client_id"][0], refreshTokenValue)
// 			}

// 			if clientSecret == r.PostForm["client_secret"][0] {
// 				log.Printf("The client secret posted is the correct value, %v\n", r.PostForm["client_secret"][0])
// 			} else {
// 				t.Errorf("The client secret posted is NOT the correct value, got %v want %v\n", r.PostForm["client_secret"][0], refreshTokenValue)
// 			}

// 			if "refresh_token" == r.PostForm["grant_type"][0] {
// 				log.Printf("The grant_type posted is the correct value, %v\n", r.PostForm["grant_type"][0])
// 			} else {
// 				t.Errorf("The grant_type posted is NOT the correct value, got %v want %v\n", r.PostForm["grant_type"][0], "refresh_token")
// 			}

// 			// Return the refreshed token values, and the other values Gitlab returns from the test HTTP server
// 			w.Header().Set("Content-Type", "application/json")

// 			responseData := tokenResponse{
// 				AccessToken:  refreshedAccessTokenValue,
// 				Type:         "bearer",
// 				ExpiresIn:    7200,
// 				RefreshToken: refreshedRefreshTokenValue,
// 				Scope:        "api",
// 				CreatedAt:    refreshedTokenCreationTime,
// 			}

// 			err = json.NewEncoder(w).Encode(&responseData)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 		}
// 	}))
// 	defer srv.Close()

// 	// Initialise dummy token store
// 	var myRefresherTokenStore RefresherTokenStore = &DummyAdapter{}

// 	// Create a refresh and access token in our dummy token store with the pre-refresh token values
// 	err := myRefresherTokenStore.SetAccessToken(ctx, models.AuthToken{
// 		ID:        tokenID,
// 		Value:     accessTokenValue,
// 		ExpiresAt: time.Now().Add(time.Minute * 5),
// 		TokenURL:  srv.URL,
// 		Type:      models.AccessTokenType,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = myRefresherTokenStore.SetRefreshToken(ctx, models.AuthToken{
// 		ID:    tokenID,
// 		Value: refreshTokenValue,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Refresh tokens expiring in the next 5 minutes
// 	err = refreshExpiringTokens(ctx, myRefresherTokenStore, clientID, clientSecret, 5)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Get the newly refreshed access and refresh tokens and ensure they contain the expected post-refresh values
// 	myNewAccessToken, err := myRefresherTokenStore.GetAccessToken(ctx, tokenID)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	myNewRefreshToken, err := myRefresherTokenStore.GetRefreshToken(ctx, tokenID)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if refreshedAccessTokenValue == myNewAccessToken.Value {
// 		log.Printf("The new access token is the correct value, %v\n", myNewAccessToken.Value)
// 	} else {
// 		t.Errorf("The new access token received is NOT the correct value, got %v want %v\n", myNewAccessToken.Value, refreshedAccessTokenValue)
// 	}

// 	if srv.URL == myNewAccessToken.TokenURL {
// 		log.Printf("The new access token URL is the correct value, %v\n", myNewAccessToken.TokenURL)
// 	} else {
// 		t.Errorf("The new access token URL received is NOT the correct value, got %v want %v\n", myNewAccessToken.TokenURL, srv.URL)
// 	}

// 	if myNewAccessToken.Type == models.AccessTokenType {
// 		log.Printf("The new access token type is the correct value, %v\n", myNewAccessToken.Type)
// 	} else {
// 		t.Errorf("The new access token URL received is NOT the correct value, got %v want %v\n", myNewAccessToken.Type, models.AccessTokenType)
// 	}

// 	if refreshedTokenCreationTime+7200 == myNewAccessToken.ExpiresAt.Unix() {
// 		log.Printf("The new access token expiration time is the correct value, %v\n", myNewAccessToken.ExpiresAt.Unix())
// 	} else {
// 		t.Errorf("The new access token expiration time received is NOT the correct value, got %v want %v\n", myNewAccessToken.ExpiresAt.Unix(), refreshedTokenCreationTime+7200)
// 	}

// 	if refreshedRefreshTokenValue == myNewRefreshToken.Value {
// 		log.Printf("The new refresh token is the correct value, %v\n", myNewRefreshToken.Value)
// 	} else {
// 		t.Errorf("The new refresh token received is NOT the correct value, got %v want %v\n", myNewRefreshToken.Value, refreshedRefreshTokenValue)
// 	}
// }

// func TestRefreshExpiringTokensKeycloak(t *testing.T) {

// 	log.Printf("Testing Keycloak access token refresh")

// 	// Set dummy values for the 'existing' access and refresh tokens, and the oauth client id and secret
// 	tokenID := "rNDSNs005xrNvrgKZ5vJGCDqwA3VQ1MB"
// 	refreshTokenValue := "QG2RX43C81P5SNS1GACEMNKVT3SDBS"
// 	accessTokenValue := "C1SB4BC3HTP841TGVS4R4G5JEAVQT4W"
// 	clientID := "iPG5UPqrV6LiXiziLbj0CBGbDvWdPWwG"
// 	clientSecret := "9p9KBXSUj037qkR55mdS0yAAecBxbb8Q"

// 	// Set the dummy values we want the access and refresh tokens to have after refreshing them.
// 	refreshedAccessTokenValue := "6XGQJCST3BY1BZ7X5X78X2MLF0W1AUB5"
// 	refreshedRefreshTokenValue := "5EU358RBY51B88OP0JJ5S15WPSTCSCX3"
// 	refreshedTokenCreationTime := time.Now().Unix()

// 	// Set up test HTTP server that refresh requests will be sent to
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method == "POST" {
// 			log.Println(w, "Received POST request!")
// 			err := r.ParseForm()
// 			if err != nil {
// 				http.Error(w, err.Error(), http.StatusBadRequest)
// 				t.Fatal(err)
// 			}

// 			// Ensure the expected values are received by the test HTTP server
// 			if refreshTokenValue == r.PostForm["refresh_token"][0] {
// 				log.Printf("The refresh token posted is the correct value, %v\n", r.PostForm["refresh_token"][0])
// 			} else {
// 				t.Errorf("The refresh token posted is NOT the correct value, got %v want %v\n", r.PostForm["refresh_token"][0], refreshTokenValue)
// 			}

// 			if clientID == r.PostForm["client_id"][0] {
// 				log.Printf("The client ID posted is the correct value, %v\n", r.PostForm["client_id"][0])
// 			} else {
// 				t.Errorf("The client ID posted is NOT the correct value, got %v want %v\n", r.PostForm["client_id"][0], refreshTokenValue)
// 			}

// 			if clientSecret == r.PostForm["client_secret"][0] {
// 				log.Printf("The client secret posted is the correct value, %v\n", r.PostForm["client_secret"][0])
// 			} else {
// 				t.Errorf("The client secret posted is NOT the correct value, got %v want %v\n", r.PostForm["client_secret"][0], refreshTokenValue)
// 			}

// 			if "refresh_token" == r.PostForm["grant_type"][0] {
// 				log.Printf("The grant_type posted is the correct value, %v\n", r.PostForm["grant_type"][0])
// 			} else {
// 				t.Errorf("The grant_type posted is NOT the correct value, got %v want %v\n", r.PostForm["grant_type"][0], "refresh_token")
// 			}

// 			// Return the refreshed token values, and the other values Keycloak returns from the test HTTP server
// 			w.Header().Set("Content-Type", "application/json")

// 			responseData := tokenResponse{
// 				AccessToken:           refreshedAccessTokenValue,
// 				Type:                  "bearer",
// 				ExpiresIn:             1800,
// 				RefreshTokenExpiresIn: 86400,
// 				RefreshToken:          refreshedRefreshTokenValue,
// 				Scope:                 "api",
// 			}

// 			err = json.NewEncoder(w).Encode(&responseData)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 		}
// 	}))
// 	defer srv.Close()

// 	// Initialise dummy token store
// 	var myRefresherTokenStore RefresherTokenStore = &DummyAdapter{}

// 	// Create a refresh and access token in our dummy token store with the pre-refresh token values
// 	err := myRefresherTokenStore.SetAccessToken(ctx, models.AuthToken{
// 		ID:        tokenID,
// 		Value:     accessTokenValue,
// 		ExpiresAt: time.Now().Add(time.Minute * 5),
// 		TokenURL:  srv.URL,
// 		Type:      models.AccessTokenType,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = myRefresherTokenStore.SetRefreshToken(ctx, models.AuthToken{
// 		ID:    tokenID,
// 		Value: refreshTokenValue,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Refresh tokens expiring in the next 5 minutes
// 	err = refreshExpiringTokens(ctx, myRefresherTokenStore, clientID, clientSecret, 5)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Get the newly refreshed access and refresh tokens and ensure they contain the expected post-refresh values
// 	myNewAccessToken, err := myRefresherTokenStore.GetAccessToken(ctx, tokenID)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	myNewRefreshToken, err := myRefresherTokenStore.GetRefreshToken(ctx, tokenID)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if refreshedAccessTokenValue == myNewAccessToken.Value {
// 		log.Printf("The new access token is the correct value, %v\n", myNewAccessToken.Value)
// 	} else {
// 		t.Errorf("The new access token received is NOT the correct value, got %v want %v\n", myNewAccessToken.Value, refreshedAccessTokenValue)
// 	}

// 	if srv.URL == myNewAccessToken.TokenURL {
// 		log.Printf("The new access token URL is the correct value, %v\n", myNewAccessToken.TokenURL)
// 	} else {
// 		t.Errorf("The new access token URL received is NOT the correct value, got %v want %v\n", myNewAccessToken.TokenURL, srv.URL)
// 	}

// 	if models.AccessTokenType == models.AccessTokenType {
// 		log.Printf("The new access token type is the correct value, %v\n", myNewAccessToken.Type)
// 	} else {
// 		t.Errorf("The new access token URL received is NOT the correct value, got %v want %v\n", myNewAccessToken.Type, models.AccessTokenType)
// 	}

// 	if refreshedTokenCreationTime+1800 == myNewAccessToken.ExpiresAt.Unix() {
// 		log.Printf("The new access token expiration time is the correct value, %v\n", myNewAccessToken.ExpiresAt.Unix())
// 	} else {
// 		t.Errorf("The new access token expiration time received is NOT the correct value, got %v want %v\n", myNewAccessToken.ExpiresAt.Unix(), refreshedTokenCreationTime+7200)
// 	}

// 	if refreshedRefreshTokenValue == myNewRefreshToken.Value {
// 		log.Printf("The new refresh token is the correct value, %v\n", myNewRefreshToken.Value)
// 	} else {
// 		t.Errorf("The new refresh token received is NOT the correct value, got %v want %v\n", myNewRefreshToken.Value, refreshedRefreshTokenValue)
// 	}

// 	if refreshedTokenCreationTime+86400 == myNewRefreshToken.ExpiresAt.Unix() {
// 		log.Printf(
// 			"The new refresh token expiration time is the correct value, %v\n",
// 			myNewRefreshToken.ExpiresAt.Unix(),
// 		)
// 	} else {
// 		t.Errorf("The new refresh token received is NOT the correct value, got %v want %v\n", myNewRefreshToken.ExpiresAt.Unix(), refreshedTokenCreationTime+86400)
// 	}
// }
