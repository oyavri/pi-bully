package config

import "time"

type WorkerConfig struct {
	RPCTimeout           time.Duration
	LeaseRenewalInterval time.Duration
	OutputBaseURI        string
}

func loadWorkerConfig() WorkerConfig {
	return WorkerConfig{
		RPCTimeout:           getDurationEnv("WORKER_RPC_TIMEOUT", 5*time.Second),
		LeaseRenewalInterval: getDurationEnv("WORKER_LEASE_RENEWAL_INTERVAL", 30*time.Second),
		OutputBaseURI:        getEnvOrDefault("WORKER_OUTPUT_BASE_URI", "s3://bully/output/"),
	}
}
