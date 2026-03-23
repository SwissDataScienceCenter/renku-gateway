package config

type PostgresConfig struct {
	Host     string
	Database string
	Username string
	Password RedactedString
}
