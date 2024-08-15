package config

import "fmt"

type RedisConfig struct {
	Type       string
	Addresses  []string
	IsSentinel bool
	Password   RedactedString
	MasterName string
	DBIndex    int
}

const DBTypeRedis string = "redis"
const DBTypeRedisMock string = "redis-mock"

func (c RedisConfig) Validate(e RunningEnvironment) error {
	if e != Development && c.Type == DBTypeRedisMock {
		return fmt.Errorf("redis type cannot be \"redis-mock\" in production")
	}
	return nil
}
