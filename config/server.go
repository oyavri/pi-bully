package config

import "os"

type ServerConfig struct {
	Port string
}

func loadServerConfig() ServerConfig {
	return ServerConfig{
		Port: os.Getenv("GRPC_PORT"),
	}
}
