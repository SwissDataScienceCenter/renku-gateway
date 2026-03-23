package utils

import (
	"fmt"
	"net/url"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
)

func GetPostgresURL(config config.PostgresConfig) (postgresURL string, err error) {
	url, err := url.Parse(fmt.Sprintf("postgres://%s:%s@%s:5432/%s", config.Username, string(config.Password), config.Host, config.Database))
	if err != nil {
		return postgresURL, err
	}
	return url.String(), nil
}
