package config

import (
	"os"
	"strconv"
)

type NodeConfig struct {
	ID      uint64
	Address string
}

func loadNodeConfig() NodeConfig {
	cfg := NodeConfig{}
	if v := os.Getenv("NODE_ID"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 64)
		cfg.ID = id
	}
	if v := os.Getenv("NODE_ADDRESS"); v != "" {
		cfg.Address = v
	}
	return cfg
}
