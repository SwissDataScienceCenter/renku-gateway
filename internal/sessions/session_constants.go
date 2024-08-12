package sessions

// Note the UI and CLI depend on some of these values, changing them will cause breaking changes
const (
	SessionCookieName = "_renku_session"
	SessionCtxKey     = "renku_session"
)

// const SessionHeaderKey = "Renku-Session"
// const CliSessionCookieName = "_renku_cli_session"
// const CliSessionCtxKey = "renku_cli_session"
// const CliSessionHeaderKey = "Renku-Cli-Session"

const (
	AccessTokenCtxKey  = "access_token"
	RefreshTokenCtxKey = "refresh_token"
	IDTokenCtxKey      = "id_token"
)
