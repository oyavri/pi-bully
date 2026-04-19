package config

import "time"

type SchedulerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	Strategy     string
}

func loadSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		PollInterval: getDurationEnv("SCHEDULER_POLL_INTERVAL", 5*time.Second),
		BatchSize:    getIntEnv("SCHEDULER_BATCH_SIZE", 10),
		Strategy:     getEnvOrDefault("SCHEDULER_STRATEGY", "round_robin"),
	}
}
