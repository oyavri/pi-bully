package config

import (
	"fmt"
	"os"
)

type Config struct {
	Database DatabaseConfig
	Log      LogConfig
	Node     NodeConfig
	Server   ServerConfig
}

func Load() (Config, error) {
	cfg := defaults()
	cfg.Node = loadNodeConfig()
	cfg.Log = loadLogConfig()
	cfg.Server = loadServerConfig()

	db, err := loadDatabaseConfig()
	if err != nil {
		return cfg, err
	}
	cfg.Database = db

	return cfg, cfg.validate()
}

func defaults() Config {
	return Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func (c *Config) validate() error {
	if c.Node.ID == 0 {
		return fmt.Errorf("NODE_ID is required and must be > 0")
	}

	if c.Node.Address == "" {
		return fmt.Errorf("NODE_ADDRESS is required")
	}

	if c.Server.Port == "" {
		return fmt.Errorf("GRPC_PORT is required")
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
