package metrics

import (
	"crypto/md5"
	"encoding/hex"

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

func NewPosthogClient(apiKey string, host string, environment string) (*PosthogMetricsClient, error) {
	client, err := posthog.NewWithConfig(
		apiKey,
		posthog.Config{
			Endpoint:               host,
			DefaultEventProperties: posthog.Properties{"environment": environment},
		},
	)
	if err != nil {
		return &PosthogMetricsClient{}, err
	}

	return &PosthogMetricsClient{posthogClient: client}, nil
}
