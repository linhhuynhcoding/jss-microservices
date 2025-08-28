package config

import (
	"github.com/linhhuynhcoding/jss-microservices/product/consts"
	"github.com/spf13/viper"
)

type Config struct {
	DBSource string `mapstructure:"DB_SOURCE"`

	CloudinaryConfig struct {
		ConnectString string `mapstructure:"CLOUDINARY_URL"`
		CloudName     string `mapstructure:"CLOUDINARY_NAME"`
		APIKey        string `mapstructure:"CLOUDINARY_API_KEY"`
		APISecret     string `mapstructure:"CLOUDINARY_API_SECRET"`
		UploadFolder  string `mapstructure:"CLOUDINARY_UPLOAD_FOLDER"`
	} `mapstructure:"CLOUDINARY"`

	UploadFolder string `mapstructure:"UPLOAD_FOLDER"`

	MarketServiceUrl string `mapstructure:"MARKET_SERVICE_URL"`
}

func NewConfig() Config {
	config, err := LoadConfig("./")
	if err != nil {
		panic(err)
	}
	return config
}

func LoadDefaultConfig(cfg *Config) {
	cfg.UploadFolder = consts.DEAFAULT_UPLOAD_FOLDER
}

func LoadConfig(path string) (config Config, err error) {
	LoadDefaultConfig(&config)

	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
