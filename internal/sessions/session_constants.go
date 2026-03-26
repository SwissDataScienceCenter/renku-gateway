package sessions

// NOTE: The UI may depend on some of these values, changing them will cause breaking changes
const (
	SessionCookieName = "_renku_session"
)

const (
	SessionCtxKey          = "renku_session"
	UserLastActivityCtxKey = "user_last_activity"
	AccessTokenCtxKey      = "access_token"
	RefreshTokenCtxKey     = "refresh_token"
	IDTokenCtxKey          = "id_token"
)
