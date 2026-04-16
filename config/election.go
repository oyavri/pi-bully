package config

import "time"

type ElectionConfig struct {
	ElectionTimeout time.Duration
}

func loadElectionConfig() ElectionConfig {
	return ElectionConfig{
		ElectionTimeout: getDurationEnv("ELECTION_TIMEOUT", 5*time.Second),
	}
}
