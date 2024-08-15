package sessions

// Note the UI may depend on some of these values, changing them will cause breaking changes
const (
	SessionCookieName = "_renku_session"
	SessionCtxKey     = "renku_session"
)

const (
	AccessTokenCtxKey  = "access_token"
	RefreshTokenCtxKey = "refresh_token"
	IDTokenCtxKey      = "id_token"
)
