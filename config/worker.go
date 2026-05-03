package config

import "time"

type WorkerConfig struct {
	RPCTimeout           time.Duration
	LeaseRenewalInterval time.Duration
}

func loadWorkerConfig() WorkerConfig {
	return WorkerConfig{
		RPCTimeout:           getDurationEnv("RPC_TIMEOUT", 5*time.Second),
		LeaseRenewalInterval: getDurationEnv("WORKER_LEASE_RENEWAL_INTERVAL", 30*time.Second),
	}
}
