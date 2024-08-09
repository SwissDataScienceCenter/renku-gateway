package tokenrefresher

import "github.com/SwissDataScienceCenter/renku-gateway/internal/models"

type RefresherSessionStore interface {
	models.SessionGetter
}
