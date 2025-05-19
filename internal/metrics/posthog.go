package metrics

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/config"
	"github.com/posthog/posthog-go"
)

type PosthogMetricsClient struct {
	posthogClient posthog.Client
}

func (p *PosthogMetricsClient) anonymizeUser(userId string) string {
	hash := md5.Sum([]byte(userId))
	return hex.EncodeToString(hash[:])
}

func (p *PosthogMetricsClient) UserLoggedIn(userId string) error {
	return p.posthogClient.Enqueue(posthog.Capture{DistinctId: p.anonymizeUser(userId), Event: "user_logged_in"})
}

func (p *PosthogMetricsClient) Close() {
	p.posthogClient.Close()
}

func NewPosthogClient(c config.PosthogConfig) (*PosthogMetricsClient, error) {
	if !c.Enabled {
		return nil, nil
	}
	client, err := posthog.NewWithConfig(
		string(c.ApiKey),
		posthog.Config{
			Endpoint:               c.Host,
			DefaultEventProperties: posthog.Properties{"environment": c.Environment},
		},
	)
	if err != nil {
		return nil, err
	}

	return &PosthogMetricsClient{posthogClient: client}, nil
}
