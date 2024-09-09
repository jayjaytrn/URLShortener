package config

import "flag"

var Config struct {
	ListenAddr   string
	ShortURLBase string
}

func SetArgs() {
	flag.StringVar(&Config.ListenAddr, "a", "localhost:8080", "server listen address")
	flag.StringVar(&Config.ShortURLBase, "b", "http://localhost:8080", "short URL base")
}
