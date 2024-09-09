package config

import (
	"flag"
	"regexp"
)

var Config struct {
	ListenAddr   string
	ShortURLBase string
}

func SetArgs() {
	flag.Func("a", "server listen address", func(s string) error {
		valid := validateServerListenAddress(s)
		if valid {
			Config.ListenAddr = s
			return nil

		}
		Config.ListenAddr = "localhost:8080"
		return nil
	})
	flag.StringVar(&Config.ShortURLBase, "b", "http://localhost:8080", "short URL base")
}

func validateServerListenAddress(listenAddress string) bool {
	r := regexp.MustCompile(`^([a-zA-Z0-9.-]+):([0-9]{1,5})$`)
	return r.MatchString(listenAddress)
}
