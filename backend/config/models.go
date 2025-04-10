package config

type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
	DbHost     string `mapstructure:"DB_HOST"`
	DbPort     string `mapstructure:"DB_PORT"`
	DbUser     string `mapstructure:"DB_USER"`
	DbPassword string `mapstructure:"DB_PASSWORD"`
	DbName     string `mapstructure:"DB_NAME"`
	DbSSLMode  string `mapstructure:"DB_SSL_MODE"`
}
