package models

import (
	"fmt"
)

type AuthTokenSet struct {
	AccessToken  AuthToken
	RefreshToken AuthToken
	IDToken      AuthToken
}

func (s *AuthTokenSet) ValidateTokensType() error {
	if s.AccessToken.ID != s.RefreshToken.ID || s.AccessToken.ID != s.IDToken.ID {
		return fmt.Errorf("tokens in a set should have the same ID")
	}
	if s.AccessToken.ProviderID != s.RefreshToken.ProviderID || s.AccessToken.ProviderID != s.IDToken.ProviderID {
		return fmt.Errorf("tokens in a set should have the same provider ID")
	}
	if s.AccessToken.Type != AccessTokenType {
		return fmt.Errorf("invalid type %s for access token %s", s.AccessToken.Type, s.AccessToken.ID)
	}
	if s.RefreshToken.Type != RefreshTokenType {
		return fmt.Errorf("invalid type %s for refresh token %s", s.RefreshToken.Type, s.RefreshToken.ID)
	}
	if s.IDToken.Type != IDTokenType {
		return fmt.Errorf("invalid type %s for ID token %s", s.IDToken.Type, s.IDToken.ID)
	}
	return nil
}
