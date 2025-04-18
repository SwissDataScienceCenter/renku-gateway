package config

type Config struct {
	RunningEnvironment
	DebugMode  bool
	Server     ServerConfig
	Sessions   SessionConfig
	Revproxy   RevproxyConfig
	Login      LoginConfig
	Redis      RedisConfig
	Posthog    PosthogConfig
	Monitoring MonitoringConfig
}

type RunningEnvironment string

const Development RunningEnvironment = "development"
const Production RunningEnvironment = "production"

func (c Config) Validate() error {
	err := c.Sessions.Validate(c.RunningEnvironment)
	if err != nil {
		return err
	}
	err = c.Login.Validate(c.RunningEnvironment)
	if err != nil {
		return err
	}
	err = c.Revproxy.Validate()
	if err != nil {
		return err
	}
	err = c.Redis.Validate(c.RunningEnvironment)
	if err != nil {
		return err
	}
	return nil
}
