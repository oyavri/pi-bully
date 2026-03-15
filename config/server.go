package config

import (
	"fmt"
	"os"
)

type ServerConfig struct {
	Port string
}

func loadServerConfig() (ServerConfig, error) {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		return ServerConfig{}, fmt.Errorf("GRPC_PORT is required")
	}

	return ServerConfig{
		Port: port,
	}, nil
}
