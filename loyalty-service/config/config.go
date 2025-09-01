package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	DBSource  string `mapstructure:"db_source"`
	MqConnStr string `mapstructure:"mq_conn_str"` //"amqp://admin:admin@localhost:5672/"
}

func NewConfig() Config {
	config, err := LoadConfig("./")
	if err != nil {
		panic(err)
	}
	return config
}

func LoadDefaultConfig(cfg *Config) {

}

func LoadConfig(path string) (config Config, err error) {
	LoadDefaultConfig(&config)

	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("looking for config in: %v\n", viper.ConfigFileUsed())
		return
	}

	err = viper.Unmarshal(&config)
	return
}
