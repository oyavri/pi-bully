package config

import "time"

type ElectionConfig struct {
	ElectionTimeout       time.Duration
	AnnounceRetryInterval time.Duration
	SendRetryInterval     time.Duration
}

func loadElectionConfig() ElectionConfig {
	return ElectionConfig{
		ElectionTimeout:       getDurationEnv("ELECTION_TIMEOUT", 5*time.Second),
		AnnounceRetryInterval: getDurationEnv("ELECTION_ANNOUNCE_RETRY_INTERVAL", 200*time.Millisecond),
		SendRetryInterval:     getDurationEnv("ELECTION_SEND_RETRY_INTERVAL", 100*time.Millisecond),
	}
}
