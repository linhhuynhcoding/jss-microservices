package config

type Config struct {
	DBSource string
}

func NewConfig() Config {
	return Config{}
}
