package config

import (
	"os"
)

type Config struct {
	Database DatabaseConfig
	Log      LogConfig
	Node     NodeConfig
	Server   ServerConfig
	Storage  StorageConfig
}

func Load() (Config, error) {
	cfg := Config{}
	cfg.Log = loadLogConfig()

	node, err := loadNodeConfig()
	if err != nil {
		return cfg, err
	}
	cfg.Node = node

	server, err := loadServerConfig()
	if err != nil {
		return cfg, err
	}
	cfg.Server = server

	db, err := loadDatabaseConfig()
	if err != nil {
		return cfg, err
	}
	cfg.Database = db

	storage, err := loadStorageConfig()
	if err != nil {
		return cfg, err
	}
	cfg.Storage = storage

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
