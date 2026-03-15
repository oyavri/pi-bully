package config

import (
	"fmt"
	"os"
	"strconv"
)

type NodeConfig struct {
	ID      uint64
	Address string
}

func loadNodeConfig() (NodeConfig, error) {
	nodeId := os.Getenv("NODE_ID")
	addr := os.Getenv("NODE_ADDRESS")

	id, err := strconv.ParseUint(nodeId, 10, 64)
	if err != nil {
		return NodeConfig{}, err
	}

	if id == 0 {
		return NodeConfig{}, fmt.Errorf("NODE_ID is required and must be > 0")
	}

	if addr == "" {
		return NodeConfig{}, fmt.Errorf("NODE_ADDRESS is required")
	}

	return NodeConfig{
		ID:      id,
		Address: addr,
	}, nil
}
