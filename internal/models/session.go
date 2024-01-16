package models

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/gwerrors"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type TokenStore interface {
	AccessTokenGetter
	AccessTokenSetter
	AccessTokenRemover
	RefreshTokenGetter
	RefreshTokenSetter
	RefreshTokenRemover
}

type SessionStore interface {
	SessionGetter
	SessionSetter
	SessionRemover
}

var randomIDGenerator IDGenerator = RandomGenerator{Length: 24}

// SessionData represents the session data that is persisted in the DB
type Session struct {
	ID   string
	Type SessionType
	// TokenIDs represent the Redis keys where the acccess and refresh tokens will be stored
	TokenIDs SerializableStringSlice
	// Mapping of state values to OIDC provider IDs
	ProviderIDs *SerializableOrderedMap
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

func (s *Session) SaveTokens(ctx context.Context, accessToken OauthToken, refreshToken OauthToken, state string) error {
	if s.tokenStore == nil {
		return fmt.Errorf("cannot save tokens when the token store is nil")
	}
	_, found := s.ProviderIDs.Delete(state)
	if !found {
		return fmt.Errorf("could not find a matching state parameter in the session")
	}
	if accessToken.ID != refreshToken.ID {
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
	return nil
}

// Equal compares if two sessions are equal ignoring the token and session store
// also the order of elements in lists or ordered maps is taken into account so that if the elements
// are the same but they are out of order then the comparison will return False
func (s *Session) Equal(other *Session) bool {
	// == does not work on some types like SerializableStringSlice or the OrderedMap
	return s.ID == other.ID &&
		s.Type == other.Type &&
		reflect.DeepEqual(s.TokenIDs, other.TokenIDs) &&
		reflect.DeepEqual(s.ProviderIDs, other.ProviderIDs) &&
		s.RedirectURL == other.RedirectURL &&
		s.CreatedAt == other.CreatedAt &&
		s.TTLSeconds == other.TTLSeconds
}

// func (s *Session) PopProviderID() string {
// 	s.loadOrCreateSessionData()
// 	if len(s.data.LoginWithProviders) == 0 {
// 		return ""
// 	}
// 	output := s.data.LoginWithProviders[0]
// 	s.data.LoginWithProviders = append(SerializableStringSlice{}, s.data.LoginWithProviders[1:]...)
// 	return output
// }

func (s *Session) PeekProviderID() string {
	pair := s.ProviderIDs.Oldest()
	if pair == nil {
		return ""
	}
	return pair.Value
}

func (s *Session) PopRedirectURL() string {
	defer func() {
		s.RedirectURL = ""
	}()
	return s.RedirectURL
}

// func (s *Session) PopOauthState() string {
// 	s.loadOrCreateSessionData()
// 	if len(s.data.OauthStates) == 0 {
// 		return ""
// 	}
// 	output := s.data.OauthStates[0]
// 	s.data.OauthStates = append(SerializableStringSlice{}, s.data.OauthStates[1:]...)
// 	return output
// }

func (s *Session) PeekOauthState() string {
	pair := s.ProviderIDs.Oldest()
	if pair == nil {
		return ""
	}
	return pair.Key
}

func (s *Session) GetAccessToken(ctx context.Context, providerID string) (OauthToken, error) {
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

// func (s *Session) AddTokenID(id string) {
// 	s.loadOrCreateSessionData()
// 	s.data.TokenIDs = append(s.data.TokenIDs, id)
// }
//
// func (s *Session) SetRedirectURL(url string) {
// 	s.loadOrCreateSessionData()
// 	s.data.RedirectURL = url
// }
//
// func (s *Session) SetProviderIDs(ids []string) {
// 	s.loadOrCreateSessionData()
// 	s.data.LoginWithProviders = ids
// }

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

func WithProviders(providerIDs ...string) SessionOption {
	return func(s *Session) error {
		if s.ProviderIDs == nil {
			providers := NewSerializableOrderedMap()
			s.ProviderIDs = &providers
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
		ProviderIDs: &providers,
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

type SessionHandlerOption func(*SessionHandler)

type SessionHandler struct {
	cookieTemplate           func() http.Cookie
	sessionTTL               time.Duration
	tokenStore               TokenStore
	sessionStore             SessionStore
	createSessionIfMissing   bool
	recreateSessionIfExpired bool
	contextKey               string
	headerKey                string
}

func (s *SessionHandler) Cookie(session *Session) *http.Cookie {
	if session == nil {
		return nil
	}
	if session.Expired() {
		return nil
	}
	cookie := s.cookieTemplate()
	cookie.Value = session.ID
	cookie.Expires = session.CreatedAt.Add(session.TTL())
	return &cookie
}

func (s *SessionHandler) Load(ctx context.Context, id string) (Session, error) {
	if s.sessionStore == nil {
		return Session{}, fmt.Errorf("cannot load a session when the session store is not defined")
	}
	session, err := s.sessionStore.GetSession(ctx, id)
	if err != nil {
		if err == redis.Nil {
			return Session{}, gwerrors.ErrSessionNotFound
		} else {
			return Session{}, err
		}
	}
	if session.Expired() {
		return Session{}, gwerrors.ErrSessionExpired
	}
	session.sessionStore = s.sessionStore
	session.tokenStore = s.tokenStore
	return session, nil
}

func (s *SessionHandler) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		createPersistSession := func(c echo.Context) error {
			session, err := NewSession(SessionWithTokenStore(s.tokenStore), SessionWithSessionStore(s.sessionStore))
			if err != nil {
				return err
			}
			err = session.Save(c.Request().Context())
			if err != nil {
				return err
			}
			c.Set(s.contextKey, session)
			c.SetCookie(s.Cookie(&session))
			return nil
		}
		return func(c echo.Context) error {
			// Check if the session ID is in the header
			// if not in the header keep going and fallback to cookie
			// the CLI will pass the session ID in the header
			headerID := c.Request().Header.Get(s.headerKey)
			if headerID != "" {
				session, err := s.Load(c.Request().Context(), headerID)
				if err != nil && err != gwerrors.ErrSessionNotFound {
					return err
				}
				c.Set(s.contextKey, session)
				return next(c)
			}
			cookie, err := c.Cookie(s.cookieTemplate().Name)
			if err != nil {
				if !(err == http.ErrNoCookie && s.createSessionIfMissing) {
					// An error other than the cookie not being found occured
					// or the cookie cannot be found but also cannot be recreated
					return err
				}
				// There is no cookie for the session, set one
				err = createPersistSession(c)
				if err != nil {
					return err
				}
				return next(c)
			}
			// A cookie was found, load the session from DB
			session, err := s.Load(c.Request().Context(), cookie.Value)
			if err != nil {
				if err != gwerrors.ErrSessionNotFound {
					return err
				}
				// The session does not exist in the DB
				if s.createSessionIfMissing {
					err = createPersistSession(c)
					if err != nil {
						return err
					}
					return next(c)
				}
			}
			if session.Expired() {
				// The session is expired
				err := session.Remove(c.Request().Context())
				if err != nil {
					return err
				}
				if s.recreateSessionIfExpired {
					err = createPersistSession(c)
					if err != nil {
						return err
					}
					return next(c)
				}
			}
			c.Set(s.contextKey, session)
			return next(c)
		}
	}
}

func WithSessionTTL(ttl time.Duration) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.sessionTTL = ttl
	}
}

// Note that the value of the cookie and expiry will be rewritten when generated
func WithCookieTemplate(cookie http.Cookie) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.cookieTemplate = func() http.Cookie {
			return cookie
		}
	}
}

func WithSessionStore(store SessionStore) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.sessionStore = store
	}
}

func WithTokenStore(store TokenStore) SessionHandlerOption {
	return func(s *SessionHandler) {
		s.tokenStore = store
	}
}

func DontCreateIfMissing() SessionHandlerOption {
	return func(s *SessionHandler) {
		s.createSessionIfMissing = false
	}
}

func DontRecreateIfExpired() SessionHandlerOption {
	return func(s *SessionHandler) {
		s.recreateSessionIfExpired = false
	}
}

func NewSessionHandler(options ...SessionHandlerOption) SessionHandler {
	store := NewDummyDBAdapter()
	sh := SessionHandler{
		recreateSessionIfExpired: true,
		createSessionIfMissing:   true,
		sessionTTL:               time.Hour,
		tokenStore:               &store,
		sessionStore:             &store,
		contextKey:               "_renku_session",
		cookieTemplate: func() http.Cookie {
			return http.Cookie{
				Name:     "_renku_session",
				Secure:   false,
				HttpOnly: true,
				Path:     "/",
				MaxAge:   3600,
			}
		},
	}
	for _, opt := range options {
		opt(&sh)
	}
	return sh
}
