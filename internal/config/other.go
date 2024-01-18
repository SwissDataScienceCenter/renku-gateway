package config

type RedisConfig struct {
	Type       string
	Addresses  []string
	IsSentinel bool
	Password   string
	MasterName string
	DBIndex    int
}

type ServerConfig struct {
	Host       string
	Port       int
	RateLimits RateLimits
	AllowOrigin []string
}

type SentryConfig struct {
	Enabled     bool
	Dsn         string
	Environment string
	SampleRate  float64
}

type PrometheusConfig struct {
	Enabled bool
	Port    int
}

type MonitoringConfig struct {
	Sentry     SentryConfig
	Prometheus PrometheusConfig
}

type RateLimits struct {
	Enabled bool
	Rate    float64
	Burst   int
}
