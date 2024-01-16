package config

import "github.com/SwissDataScienceCenter/renku-gateway/internal/models"

type Config struct {
	Server     ServerConfig
	Revproxy   RevproxyConfig
	Login      LoginConfig
	Redis      RedisConfig
	Monitoring MonitoringConfig
}

type DBAdapter interface {
	models.AccessTokenGetter
	models.AccessTokenSetter
	models.AccessTokenRemover
	models.RefreshTokenGetter
	models.RefreshTokenSetter
	models.RefreshTokenRemover
	models.SessionGetter
	models.SessionSetter
	models.SessionRemover
}

const DBTypeRedis string = "redis"
const DBTypeRedisMock string = "redis-mock"

// func (c *Config) DBAdapter() (DBAdapter, error) {
// 	var encryptor models.Encryptor
// 	var err error
// 	if c.Login.TokenEncryption.Enabled {
// 		encryptor, err = db.NewGCMEncryptor(c.Login.TokenEncryption.SecretKey)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	switch c.Redis.Type {
// 	case DBTypeRedis:
// 		if c.Redis.IsSentinel {
// 			rdb := redis.NewFailoverClient(&redis.FailoverOptions{
// 				MasterName:       c.Redis.MasterName,
// 				SentinelAddrs:    c.Redis.Addresses,
// 				Password:         c.Redis.Password,
// 				DB:               c.Redis.DBIndex,
// 				SentinelPassword: c.Redis.Password,
// 			})
// 			return db.NewRedisAdapter(rdb, encryptor), nil
// 		}
// 		rdb := redis.NewClient(&redis.Options{
// 			Password: c.Redis.Password,
// 			DB:       c.Redis.DBIndex,
// 			Addr:     c.Redis.Addresses[0],
// 		})
// 		return db.NewRedisAdapter(rdb, encryptor), nil
// 	case DBTypeRedisMock:
// 		return db.NewMockRedisAdapter(), nil
// 	default:
// 		return nil, fmt.Errorf("unrecognized persistence type %v", c.Redis.Type)
// 	}
// }
