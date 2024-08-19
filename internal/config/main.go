package config

import (
	"fmt"
)

type RedactedString string

func (r RedactedString) String() string {
	return fmt.Sprintf("<redacted-%d-chars>", len(r))
}

func (r RedactedString) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", r.String())), nil
}

func (r RedactedString) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r RedactedString) MarshalBinary() ([]byte, error) {
	return []byte(r.String()), nil
}

type Config struct {
	RunningEnvironment
	Server     ServerConfig
	Sessions   SessionConfig
	Revproxy   RevproxyConfig
	Login      LoginConfig
	Redis      RedisConfig
	Monitoring MonitoringConfig
}

type RunningEnvironment string

const Development RunningEnvironment = "development"
const Production RunningEnvironment = "production"

func (c Config) Validate() error {
	err := c.Sessions.Validate()
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
