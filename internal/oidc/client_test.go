package oidc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
)

type mockRelyingParty struct {
	isPKCE   bool
	tokenURL string
}

func (m mockRelyingParty) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID: "mock-client",
		Endpoint: oauth2.Endpoint{TokenURL: m.tokenURL},
	}
}

func (m mockRelyingParty) Issuer() string {
	return "https://token.url"
}

func (m mockRelyingParty) IsPKCE() bool {
	return m.isPKCE
}

func (mockRelyingParty) CookieHandler() *httphelper.CookieHandler {
	return nil
}

func (mockRelyingParty) HttpClient() *http.Client {
	return http.DefaultClient
}

func (mockRelyingParty) IsOAuth2Only() bool {
	return true
}

func (mockRelyingParty) Signer() jose.Signer {
	return nil
}

func (mockRelyingParty) GetEndSessionEndpoint() string {
	return ""
}

func (mockRelyingParty) GetRevokeEndpoint() string {
	return ""
}

func (mockRelyingParty) UserinfoEndpoint() string {
	return ""
}

func (mockRelyingParty) GetDeviceAuthorizationEndpoint() string {
	return ""
}

func (mockRelyingParty) IDTokenVerifier() rp.IDTokenVerifier {
	return nil
}

func (mockRelyingParty) ErrorHandler() func(http.ResponseWriter, *http.Request, string, string, string) {
	return func(http.ResponseWriter, *http.Request, string, string, string) {}
}

func newMockRelyingParty(tokenURL string) rp.RelyingParty {
	return mockRelyingParty{isPKCE: true, tokenURL: tokenURL}
}

func TestClientReturnsID(t *testing.T) {
	client := oidcClient{
		client: newMockRelyingParty("https://token.url"),
		id:     "id",
	}
	assert.Equal(t, "id", client.getID())
}

type TestTokenCallbackScenario struct {
	Name                 string
	Error                error
	ProviderID           string
	AccessToken          string
	RefreshToken         string
	IDToken              string
	State                string
	AccessTokenExpiresIn int
	Now                  time.Time
	TokenURL             string
}

func TestTokenCallback(t *testing.T) {
	testCases := []TestTokenCallbackScenario{
		{
			Name:                 "regular",
			ProviderID:           "id",
			AccessToken:          "accessToken",
			RefreshToken:         "refreshToken",
			IDToken:              "idToken",
			State:                "state",
			AccessTokenExpiresIn: 50,
			Now:                  time.Now(),
			TokenURL:             "https://token.url",
		},
		{
			Name:  "error",
			Error: fmt.Errorf("Some error"),
		},
	}

	parametrizedTest := func(testCase TestTokenCallbackScenario) func(*testing.T) {
		return func(t *testing.T) {
			client := oidcClient{
				client: newMockRelyingParty(testCase.TokenURL),
				id:     testCase.ProviderID,
			}
			oauth2Token := &oauth2.Token{
				AccessToken:  testCase.AccessToken,
				RefreshToken: testCase.RefreshToken,
				Expiry:       testCase.Now.Add(time.Second * time.Duration(testCase.AccessTokenExpiresIn)),
			}
			tokens := oidc.Tokens[*oidc.IDTokenClaims]{
				Token:         oauth2Token,
				IDTokenClaims: &oidc.IDTokenClaims{},
				IDToken:       testCase.IDToken,
			}
			tokenCallback := func(tokenSet models.AuthTokenSet) error {
				if testCase.Error != nil {
					return testCase.Error
				}
				assert.Equal(t, testCase.AccessToken, tokenSet.AccessToken.Value)
				assert.Equal(t, models.AccessTokenType, tokenSet.AccessToken.Type)
				assert.Equal(t, testCase.ProviderID, tokenSet.AccessToken.ProviderID)
				assert.Equal(t, testCase.TokenURL, tokenSet.AccessToken.TokenURL)
				assert.Equal(t, testCase.RefreshToken, tokenSet.RefreshToken.Value)
				assert.Equal(t, models.RefreshTokenType, tokenSet.RefreshToken.Type)
				assert.Equal(t, testCase.ProviderID, tokenSet.RefreshToken.ProviderID)
				assert.Equal(t, testCase.TokenURL, tokenSet.RefreshToken.TokenURL)
				assert.Equal(
					t,
					testCase.Now.Add(time.Second*time.Duration(testCase.AccessTokenExpiresIn)).Unix(),
					tokenSet.AccessToken.ExpiresAt.Unix(),
				)
				return nil
			}
			codeExchangeCallback := client.getCodeExchangeCallback(tokenCallback)
			rec := httptest.NewRecorder()
			codeExchangeCallback(rec, httptest.NewRequest("GET", "/", nil), &tokens, "state", client.client)
			if testCase.Error != nil {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			} else {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			}
		}
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, parametrizedTest(testCase))
	}

}
