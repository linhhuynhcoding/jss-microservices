package config

type Config struct {
	DBSource  string `mapstructure:"db_source"`
	MqConnStr string `mapstructure:"mq_conn_str"` //"amqp://admin:admin@localhost:5672/"
}

func NewConfig() Config {
	return Config{}
}
