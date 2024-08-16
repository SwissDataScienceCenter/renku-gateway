package authentication

import (
	"context"
	"fmt"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Authenticator map[string]tokenVerifier

func (a Authenticator) VerifyAccessToken(ctx context.Context, accessToken string) (oidc.TokenClaims, error) {
	claims := new(oidc.TokenClaims)
	payload, err := oidc.ParseToken(accessToken, claims)
	if err != nil {
		return oidc.TokenClaims{}, err
	}

	verifierID := claims.AuthorizedParty
	verifier, ok := a[verifierID]
	if !ok {
		return oidc.TokenClaims{}, fmt.Errorf("token has an unrecognized authorized party %s", verifierID)
	}
	return verifier.verifyAccessToken(ctx, accessToken, payload, claims)
}

type AuthenticatorOption func(Authenticator) error

func WithConfig(config []config.AuthorizationVerifier) AuthenticatorOption {
	return func(a Authenticator) error {
		for _, verifierConfig := range config {
			tokenVerifier, err := newTokenVerifier(withConfig(verifierConfig))
			if err != nil {
				return err
			}
			a[tokenVerifier.id] = *tokenVerifier
		}
		return nil
	}
}

func NewAuthenticator(options ...AuthenticatorOption) (Authenticator, error) {
	a := Authenticator{}
	for _, opt := range options {
		err := opt(a)
		if err != nil {
			return Authenticator{}, err
		}
	}
	return a, nil
}
