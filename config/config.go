package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Database   DatabaseConfig
	Log        LogConfig
	Node       NodeConfig
	Server     ServerConfig
	Storage    StorageConfig
	Memberlist MemberlistConfig
	Election   ElectionConfig
	Health     HealthConfig
	Scheduler  SchedulerConfig
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

	ml, err := loadMemberlistConfig()
	if err != nil {
		return cfg, err
	}
	cfg.Memberlist = ml

	cfg.Election = loadElectionConfig()
	cfg.Health = loadHealthConfig()
	cfg.Scheduler = loadSchedulerConfig()

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}

	return d
}

func getIntEnv(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}

	return i
}
