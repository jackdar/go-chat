package server

import (
	"flag"
	"fmt"
)

type Config struct {
	Host string
	Port string
}

func ParseConfig() *Config {
	config := &Config{}

	flag.StringVar(&config.Host, "host", "0.0.0.0", "Host to bind to")
	flag.StringVar(&config.Port, "port", "8080", "Port to listen on")
	flag.Parse()

	return config
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
