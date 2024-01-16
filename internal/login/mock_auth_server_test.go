package login

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var testJWKSContent = map[string][]map[string]string{
	"keys": {{
		"kty": "RSA",
		"n":   "r6-teKWvC4m5HPOVT49lRBkEl0TcodXYwszKDb0fdJM9qGsaimyD-xobvXKY-sm0NwmcShAkFzUJiZUPHFgYbGUTgqdeiQ-h8SE2qEfB8pfmdGR8Tp7cwlxcsVfsYOc2cvYUt0TV6bmzqWc7IEgaOAKnQ261YmZ4FutdPckAK4NC8jjxA-3cmbft_mf4R1kSs1Ik8WAbkt8GmSXfIfjCjjEWgAN-dlwBzsR4RAPh1dY_Av8W1eiMnJymCCFDP8WTlRQ7Qn_fJ3zEJazeex9tZVXaX1VfXSDxzNDuy_sL9sQD71i3XCKdBr2gQKNSndOWHyf4bAxVR2XCM2zQbgQH1DNoAXmVGffYjTwMYgjzc-PTeJ_oSUND776l5yfqhjrMJpw3KNUSEczbUDqUIEA_mb3kr5MHs1_WX6KajIJpX516aaHEB0-_rtKVzTsjuusbBFw7oTHKFA2vXxqK3xznOZXZKlA5WsvgTLUxOaJf2_tU5LW1dVZ85QiOz8Qxj-idLp-eGxqOTPI4_TbhERaHBzAd08XjeGHDvammDdVpiBAlYeCGvOvw2VCZu7_N0Cg1Ezsg-hQbKM5ZJ35N3GWaAB_gJ7aFKm6tfD9SnZAc1Nb-Cll5Ouf2Ffvim2jNR3wHStsr4Kip5RjThejHs4oh--ix21LH_aU_uPdRWIQXzyM", // nolint: lll
		"e":   "AQAB",
		"alg": "RS256",
		"kid": "id1",
		"use": "sig",
	}},
}

const testRSAPrivateKey string = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`

const testRSAPublicKey string = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAr6+teKWvC4m5HPOVT49l
RBkEl0TcodXYwszKDb0fdJM9qGsaimyD+xobvXKY+sm0NwmcShAkFzUJiZUPHFgY
bGUTgqdeiQ+h8SE2qEfB8pfmdGR8Tp7cwlxcsVfsYOc2cvYUt0TV6bmzqWc7IEga
OAKnQ261YmZ4FutdPckAK4NC8jjxA+3cmbft/mf4R1kSs1Ik8WAbkt8GmSXfIfjC
jjEWgAN+dlwBzsR4RAPh1dY/Av8W1eiMnJymCCFDP8WTlRQ7Qn/fJ3zEJazeex9t
ZVXaX1VfXSDxzNDuy/sL9sQD71i3XCKdBr2gQKNSndOWHyf4bAxVR2XCM2zQbgQH
1DNoAXmVGffYjTwMYgjzc+PTeJ/oSUND776l5yfqhjrMJpw3KNUSEczbUDqUIEA/
mb3kr5MHs1/WX6KajIJpX516aaHEB0+/rtKVzTsjuusbBFw7oTHKFA2vXxqK3xzn
OZXZKlA5WsvgTLUxOaJf2/tU5LW1dVZ85QiOz8Qxj+idLp+eGxqOTPI4/TbhERaH
BzAd08XjeGHDvammDdVpiBAlYeCGvOvw2VCZu7/N0Cg1Ezsg+hQbKM5ZJ35N3GWa
AB/gJ7aFKm6tfD9SnZAc1Nb+Cll5Ouf2Ffvim2jNR3wHStsr4Kip5RjThejHs4oh
++ix21LH/aU/uPdRWIQXzyMCAwEAAQ==
-----END PUBLIC KEY-----
`

// testAuthServer is a simple oauth2 server which mocks/implements the same functionality
// provided by Keycloak and needed for testing.
type testAuthServer struct {
	Authorized   bool
	RefreshToken string
	ClientID     string
	JWTSecretKey string
	CallbackURI  string
	server       *httptest.Server
}

func (*testAuthServer) jwksEndpoint(c echo.Context) error {
	return c.JSON(http.StatusOK, testJWKSContent)
}

func (t *testAuthServer) wktEndpoint(c echo.Context) error {
	type wkt struct {
		Issuer                string   `json:"issuer,omitempty"`
		AuthorizationEndpoint string   `json:"authorization_endpoint,omitempty"`
		TokenEndpoint         string   `json:"token_endpoint,omitempty"`
		JWKSUri               string   `json:"jwks_uri,omitempty"`
		ResponseTypesSup      []string `json:"response_types_supported,omitempty"`
		SubjectTypes          []string `json:"subject_types,omitempty"`
		IdTokenSignAlgs       []string `json:"id_token_signing_alg_values_supported,omitempty"`
	}
	res := wkt{
		Issuer:                t.Server().URL,
		AuthorizationEndpoint: t.Server().URL + "/authorize",
		TokenEndpoint:         t.Server().URL + "/token",
		JWKSUri:               t.Server().URL + "/jwks",
		ResponseTypesSup:      []string{"code", "id_token", "token id_token"},
		SubjectTypes:          []string{"public"},
		IdTokenSignAlgs:       []string{"RS256"},
	}
	return c.JSON(http.StatusOK, res)
}

func (t *testAuthServer) getJWT() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		"aud": t.ClientID, // oauth2 proxy providers want the aud to match the clientID
		"sub": "sub",
		"iss": t.Server().URL,
		"iat": time.Now().Unix(),
	})
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(testRSAPrivateKey))
	if err != nil {
		return "", err
	}
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (t *testAuthServer) authorizeEndpoint(c echo.Context) error {
	if t.Authorized {
		state := c.QueryParam("state")
		redirectURL := c.QueryParam("redirect_uri")
		vals := url.Values{}
		vals.Add("code", "codeValue")
		vals.Add("state", state)
		return c.Redirect(http.StatusFound, redirectURL+"?"+vals.Encode())
	}
	return c.String(http.StatusUnauthorized, "not authorized by test auth server")
}

func (t *testAuthServer) tokenEndpoint(c echo.Context) error {
	if t.Authorized {
		jwtToken, err := t.getJWT()
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]string{
			"access_token":  jwtToken,
			"refresh_token": t.RefreshToken,
			"id_token":      jwtToken,
		})
	}
	return c.String(http.StatusBadRequest, "bad request")
}

func (t *testAuthServer) Server() *httptest.Server {
	if t.server == nil {
		panic("Server has not been started")
	}
	return t.server
}

func (t *testAuthServer) Start() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/authorize", t.authorizeEndpoint)
	e.GET("/jwks", t.jwksEndpoint)
	e.POST("/token", t.tokenEndpoint)
	e.GET("/.well-known/openid-configuration", t.wktEndpoint)
	t.server = httptest.NewServer(e.Server.Handler)
}

func (t *testAuthServer) ProviderConfig() config.OIDCClient {
	return config.OIDCClient{
		Default:               true,
		Issuer:                t.Server().URL,
		ClientID:              t.ClientID,
		ClientSecret:          "client-secret-value",
		CallbackURI:           t.CallbackURI,
		UnsafeNoCookieHandler: true,
	}
}
