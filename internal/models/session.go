package models

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
)

type TokenStore interface {
	AccessTokenGetter
	AccessTokenSetter
	AccessTokenRemover
	RefreshTokenGetter
	RefreshTokenSetter
	RefreshTokenRemover
	IDTokenGetter
	IDTokenSetter
	IDTokenRemover
}

type SessionStore interface {
	SessionGetter
	SessionSetter
	SessionRemover
}

// Note the UI and CLI depend on some of these values, changing them will cause breaking changes
const SessionCookieName = "_renku_session"
const SessionCtxKey = "_renku_session"
const SessionHeaderKey = "Renku-Session"
const CliSessionCookieName = "_renku_cli_session"
const CliSessionCtxKey = "_renku_cli_session"
const CliSessionHeaderKey = "Renku-Cli-Session"
// BasicAuthUsername is used as the username in the Basic Auth authorization header when the CLI
// sends session IDs for Git requests.
const BasicAuthUsername = "Renku-Session"

var randomIDGenerator IDGenerator = RandomGenerator{Length: 24}

// SessionData represents the session data that is persisted in the DB
type Session struct {
	ID   string
	Type SessionType
	// TokenIDs represent the Redis keys where the acccess and refresh tokens will be stored
	TokenIDs SerializableStringSlice
	// Mapping of state values to OIDC provider IDs
	ProviderIDs SerializableOrderedMap
	// The url to redirect to when the login flow is complete (i.e. Renku homepage)
	RedirectURL string
	// UTC timestamp for when the session was created
	CreatedAt    time.Time
	TTLSeconds   SerializableInt
	tokenStore   TokenStore
	sessionStore SessionStore
}

type SessionOption func(*Session) error

func (s *Session) Expired() bool {
	return time.Now().UTC().After(s.CreatedAt.Add(s.TTL()))
}

func (s *Session) TTL() time.Duration {
	return time.Duration(s.TTLSeconds) * time.Second
}

func (s *Session) SaveTokens(ctx context.Context, accessToken OauthToken, refreshToken OauthToken, idToken OauthToken, state string) error {
	if s.tokenStore == nil {
		return fmt.Errorf("cannot save tokens when the token store is nil")
	}
	if state != "" {
		_, found := s.ProviderIDs.Delete(state)
		if !found {
			return fmt.Errorf("could not find a matching state parameter in the session")
		}
	}
	if accessToken.ID != refreshToken.ID || accessToken.ID != idToken.ID {
		return fmt.Errorf("trying to save access and refresh token with different IDs")
	}
	s.TokenIDs = append(s.TokenIDs, accessToken.ID)
	err := s.Save(ctx)
	if err != nil {
		return err
	}
	err = s.tokenStore.SetAccessToken(ctx, accessToken)
	if err != nil {
		return err
	}
	err = s.tokenStore.SetRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}
	if idToken.Value != "" {
		err = s.tokenStore.SetIDToken(ctx, idToken)
		if err != nil {
			return err
		}
	}
	return nil
}

// Equal compares if two sessions are equal ignoring the token and session store
// also the order of elements in lists or ordered maps is taken into account so that if the elements
// are the same but they are out of order then the comparison will return False
func (s *Session) Equal(other *Session) bool {
	// == does not work on some types like SerializableStringSlice or the OrderedMap
	if s == nil && other == nil {
		return true
	} else if (s == nil && other != nil) || (s != nil && other == nil) {
		return false
	}
	return s.ID == other.ID &&
		s.Type == other.Type &&
		reflect.DeepEqual(s.TokenIDs, other.TokenIDs) &&
		reflect.DeepEqual(s.ProviderIDs, other.ProviderIDs) &&
		s.RedirectURL == other.RedirectURL &&
		s.CreatedAt == other.CreatedAt &&
		s.TTLSeconds == other.TTLSeconds
}

func (s *Session) PeekProviderID() string {
	pair := s.ProviderIDs.Oldest()
	if pair == nil {
		return ""
	}
	return pair.Value
}

func (s *Session) PopRedirectURL() string {
	redirectURL := s.RedirectURL
	s.RedirectURL = ""
	return redirectURL
}

func (s *Session) PeekOauthState() string {
	pair := s.ProviderIDs.Oldest()
	if pair == nil {
		return ""
	}
	return pair.Key
}

func (s *Session) GetAccessToken(ctx context.Context, providerID string) (OauthToken, error) {
	if s.tokenStore == nil {
		return OauthToken{}, fmt.Errorf("cannot get a token when the token store is not defined")
	}
	tokens, err := s.tokenStore.GetAccessTokens(ctx, s.TokenIDs...)
	if err != nil {
		return OauthToken{}, err
	}
	token, found := tokens[providerID]
	if !found {
		return OauthToken{}, gwerrors.ErrTokenNotFound
	}
	if token.Expired() {
		return OauthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (s *Session) GetIDToken(ctx context.Context, providerID string) (OauthToken, error) {
	if s.tokenStore == nil {
		return OauthToken{}, fmt.Errorf("cannot get a token when the token store is not defined")
	}
	tokens, err := s.tokenStore.GetIDTokens(ctx, s.TokenIDs...)
	if err != nil {
		return OauthToken{}, err
	}
	token, found := tokens[providerID]
	if !found {
		return OauthToken{}, gwerrors.ErrTokenNotFound
	}
	if token.Expired() {
		return OauthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (s *Session) GetRefreshToken(ctx context.Context, providerID string) (OauthToken, error) {
	if s.tokenStore == nil {
		return OauthToken{}, fmt.Errorf("cannot get a token when the token store is not defined")
	}
	tokens, err := s.tokenStore.GetRefreshTokens(ctx, s.TokenIDs...)
	if err != nil {
		return OauthToken{}, err
	}
	token, found := tokens[providerID]
	if !found {
		return OauthToken{}, gwerrors.ErrTokenNotFound
	}
	if token.Expired() {
		return OauthToken{}, gwerrors.ErrTokenExpired
	}
	return token, nil
}

func (s *Session) Save(ctx context.Context) error {
	if s.Expired() {
		return gwerrors.ErrSessionExpired
	}
	if s.sessionStore == nil {
		return fmt.Errorf("cannot save a session when the session store is not defined")
	}
	err := s.sessionStore.SetSession(ctx, *s)
	return err
}

func (s *Session) Remove(ctx context.Context) error {
	if s.sessionStore == nil {
		return fmt.Errorf("cannot remove a session when the session store is not defined")
	}
	return s.sessionStore.RemoveSession(ctx, s.ID)
}

func (s *Session) SetProviders(ctx context.Context, providerIDs ...string) error {
	err := WithProviders(providerIDs...)(s)
	if err != nil {
		return err
	}
	return s.Save(ctx)
}

func (s *Session) SetRedirectURL(ctx context.Context, url string) error {
	err := WithRedirectURL(url)(s)
	if err != nil {
		return err
	}
	return s.Save(ctx)
}

func (s *Session) SetTokenStore(tokenStore TokenStore) {
	s.tokenStore = tokenStore
}

func (s *Session) SetSessionStore(sessionStore SessionStore) {
	s.sessionStore = sessionStore
}

func WithProviders(providerIDs ...string) SessionOption {
	return func(s *Session) error {
		blank := SerializableOrderedMap{}
		if s.ProviderIDs == blank {
			providers := NewSerializableOrderedMap()
			s.ProviderIDs = providers
		}
		for _, provider := range providerIDs {
			aprovider := provider
			state, err := randomIDGenerator.ID()
			if err != nil {
				return err
			}
			s.ProviderIDs.Set(state, aprovider)
		}
		return nil
	}
}

func WithRedirectURL(url string) SessionOption {
	return func(s *Session) error {
		s.RedirectURL = url
		return nil
	}
}

func SessionWithSessionStore(store SessionStore) SessionOption {
	return func(s *Session) error {
		s.sessionStore = store
		return nil
	}
}

func SessionWithTokenStore(store TokenStore) SessionOption {
	return func(s *Session) error {
		s.tokenStore = store
		return nil
	}
}

func NewSession(options ...SessionOption) (Session, error) {
	id, err := randomIDGenerator.ID()
	if err != nil {
		return Session{}, err
	}
	providers := NewSerializableOrderedMap()
	session := Session{
		CreatedAt:   time.Now().UTC(),
		TTLSeconds:  SerializableInt((time.Hour * 8).Seconds()),
		ProviderIDs: providers,
		TokenIDs:    SerializableStringSlice{},
		ID:          id,
	}
	// Apply user specified options
	for _, opt := range options {
		err = opt(&session)
		if err != nil {
			return Session{}, err
		}
	}
	return session, nil
}

