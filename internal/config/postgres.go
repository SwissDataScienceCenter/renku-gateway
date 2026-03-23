package config

type PostgresConfig struct {
	Host     string
	Database string
	Username string
	Password RedactedString
}

func (c PostgresConfig) Validate() error {
	if c.Database == "" {
		c.Database = "renku_gateway"
	}
	return nil
}
