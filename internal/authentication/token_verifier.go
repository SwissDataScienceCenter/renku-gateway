package authentication

import (
	"context"
	"fmt"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

const verifierOffset = time.Second

type tokenVerifier struct {
	id              string
	issuer          string
	audience        string
	authorizedParty string

	keyset oidc.KeySet
}

func (tv tokenVerifier) verifyAccessToken(ctx context.Context, accessToken string, payload []byte, claims *oidc.TokenClaims) (oidc.TokenClaims, error) {
	if err := oidc.CheckSubject(claims); err != nil {
		return oidc.TokenClaims{}, err
	}

	if err := oidc.CheckIssuer(claims, tv.issuer); err != nil {
		return oidc.TokenClaims{}, err
	}

	if err := oidc.CheckAudience(claims, tv.audience); err != nil {
		return oidc.TokenClaims{}, err
	}

	if err := oidc.CheckAuthorizedParty(claims, tv.authorizedParty); err != nil {
		return oidc.TokenClaims{}, err
	}

	if err := oidc.CheckSignature(ctx, accessToken, payload, claims, []string{}, tv.keyset); err != nil {
		return oidc.TokenClaims{}, err
	}

	if err := oidc.CheckExpiration(claims, verifierOffset); err != nil {
		return oidc.TokenClaims{}, err
	}

	return *claims, nil
}

type tokenVerifierOption func(*tokenVerifier) error

func withConfig(config config.AuthorizationVerifier) tokenVerifierOption {
	discover := func(issuer string) (rp.Endpoints, error) {
		return rp.Discover(issuer, httphelper.DefaultHTTPClient)
	}
	return func(tv *tokenVerifier) error {
		tv.id = config.AuthorizedParty
		tv.issuer = config.Issuer
		tv.audience = config.Audience
		tv.authorizedParty = config.AuthorizedParty

		endpoints, err := discover(tv.issuer)
		if err != nil {
			return err
		}
		tv.keyset = rp.NewRemoteKeySet(httphelper.DefaultHTTPClient, endpoints.JKWsURL)

		return nil
	}
}

func newTokenVerifier(options ...tokenVerifierOption) (*tokenVerifier, error) {
	tv := tokenVerifier{}
	for _, opt := range options {
		err := opt(&tv)
		if err != nil {
			return &tokenVerifier{}, err
		}
	}
	if tv.id == "" {
		return &tokenVerifier{}, fmt.Errorf("token verifier id is not set")
	}
	if tv.issuer == "" {
		return &tokenVerifier{}, fmt.Errorf("token verifier issuer is not set")
	}
	if tv.audience == "" {
		return &tokenVerifier{}, fmt.Errorf("token verifier audience is not set")
	}
	if tv.authorizedParty == "" {
		return &tokenVerifier{}, fmt.Errorf("token verifier authorized party is not set")
	}
	if tv.keyset == nil {
		return &tokenVerifier{}, fmt.Errorf("token verifier key set is not initialized")
	}
	return &tv, nil
}
