package oauth

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *OAuthServer) GetProviders(c echo.Context) error {
	list := make(ProviderList, len(s.config.Applications))
	i := 0
	for id, provider := range s.config.Applications {
		list[i] = struct {
			Id       *string   "json:\"id,omitempty\""
			Provider *Provider "json:\"provider,omitempty\""
		}{
			Id: &id,
			Provider: &Provider{
				ClientId:    &provider.ClientID,
				DisplayName: &provider.DisplayName,
			},
		}
		i += 1
	}
	return c.JSON(http.StatusOK, list)
}
