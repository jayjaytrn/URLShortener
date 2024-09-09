package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

var Config struct {
	ServerAddress string `env:"SERVER_ADDRESS,required"`
	BaseURL       string `env:"BASE_URL,required"`
}

func SetArgs() {
	err := env.Parse(&Config)
	if err != nil {
		flag.StringVar(&Config.ServerAddress, "a", "localhost:8080", "server listen address")
		flag.StringVar(&Config.BaseURL, "b", "http://localhost:8080", "short URL base")
	}
}
