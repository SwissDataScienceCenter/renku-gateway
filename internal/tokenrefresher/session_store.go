package tokenrefresher

import (
	"github.com/SwissDataScienceCenter/renku-gateway/internal/sessions"
)

type RefresherSessionStore interface {
	sessions.SessionGetter
}
