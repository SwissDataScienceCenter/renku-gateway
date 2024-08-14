package login

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

const accessToken = `eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICIzQWVuYW9tb3BtUW5vYXB2YU5VUm9iajNDeDFuV01OTXhfY0lrSHBaUjVjIn0.eyJleHAiOjE3MjM2NDE4NzksImlhdCI6MTcyMzY0MDA3OSwiYXV0aF90aW1lIjoxNzIzNjE2MzE3LCJqdGkiOiIwYzI5ZWZmYi03ZmQ2LTQ0ZGQtOTRhZC1kYzliMDVhZjA4ZGIiLCJpc3MiOiJodHRwczovL3Jlbmt1LWNpLWd3LTcxMC5kZXYucmVua3UuY2gvYXV0aC9yZWFsbXMvUmVua3UiLCJhdWQiOlsicmVua3UiLCJhY2NvdW50Il0sInN1YiI6Ijk3YTMxOTk3LWU4MDAtNDI3Mi04MTZkLTkyZDE1YmFiYmU5OSIsInR5cCI6IkJlYXJlciIsImF6cCI6InJlbmt1Iiwic2Vzc2lvbl9zdGF0ZSI6ImQzM2M5NjI2LWU1OWMtNGUwMC04YTBjLTBhYzM5ZWEzY2QyNyIsImFjciI6IjAiLCJhbGxvd2VkLW9yaWdpbnMiOlsiaHR0cHM6Ly9yZW5rdS1jaS1ndy03MTAuZGV2LnJlbmt1LmNoLyoiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIm9mZmxpbmVfYWNjZXNzIiwiZGVmYXVsdC1yb2xlcy1yZW5rdSIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJvcGVuaWQgbWljcm9wcm9maWxlLWp3dCBwcm9maWxlIGVtYWlsIiwic2lkIjoiZDMzYzk2MjYtZTU5Yy00ZTAwLThhMGMtMGFjMzllYTNjZDI3IiwidXBuIjoiZmxvcmEudGhpZWJhdXRAc2RzYy5ldGh6LmNoIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJuYW1lIjoiRmxvcmEgVGhpZWJhdXQiLCJncm91cHMiOlsib2ZmbGluZV9hY2Nlc3MiLCJkZWZhdWx0LXJvbGVzLXJlbmt1IiwidW1hX2F1dGhvcml6YXRpb24iXSwicHJlZmVycmVkX3VzZXJuYW1lIjoiZmxvcmEudGhpZWJhdXRAc2RzYy5ldGh6LmNoIiwiZ2l2ZW5fbmFtZSI6IkZsb3JhIiwiZmFtaWx5X25hbWUiOiJUaGllYmF1dCIsImVtYWlsIjoiZmxvcmEudGhpZWJhdXRAc2RzYy5ldGh6LmNoIn0.gtm-j19n30OMRsU9VDoTcUN495ElnMW0H-uXM6-GnrIuMQ_GK2FN03U8a6fz8-NVRRe1BDKVbvUitCegwazm2IrWNXzFyFoCEqOUBTSofmNlNeHoQevoipT7dcbyhS0DXketzCRZrwUGdwe7NdrG2VLrPiuDX4CjBjYGWrH9ZVjtUi5Ba3l7Oc8a-BkrZOyu1hUTVd1DmVRf2McUk3fQ7XgGpxOsgmtBaFEbuzBkdNMqfV1C_zHppouRXX46BwhPq0QEeEPN0z1UXC1auhjS6AH_0hhikV-eu05XjYk5_-ngDZADeV0w09P76wxMd-cB08D0BN8H35Pc5x1vDhQINw`

func TestVerifyToken(t *testing.T) {
	client, err := rp.NewRelyingPartyOIDC(
		"https://renku-ci-gw-710.dev.renku.ch/auth/realms/Renku",
		"",
		"",
		"https://example.org",
		[]string{},
	)
	assert.NoError(t, err)
	ks := client.IDTokenVerifier().KeySet()

	// token, err := jwt.ParseSigned(accessToken)
	// assert.NoError(t, err)
	// claims := jwt.Claims{}
	// err = token.Claims(ks, &claims)
	// assert.NoError(t, err)
	// assert.Equal(t, "", claims)

	claims := new(oidc.TokenClaims)
	payload, err := oidc.ParseToken(accessToken, claims)
	assert.NoError(t, err)
	err = oidc.CheckSignature(context.Background(), accessToken, payload, claims, []string{"RS256"}, ks)
	assert.NoError(t, err)

	// if claims.SignatureAlg == "RS256" {
	// 	err = oidc.CheckSignature(ctx, val, payload, claims, []string{"RS256"}, ks)
	// 	if err != nil {
	// 		return models.AuthToken{}, err
	// 	}
	// }
	// if tokenType != models.RefreshTokenType {
	// 	err = oidc.CheckExpiration(claims, 0)
	// 	if err != nil {
	// 		return models.AuthToken{}, err
	// 	}
	// }
}
