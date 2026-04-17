package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type MemberlistConfig struct {
	BindAddr         string
	BindPort         int
	AdvertiseAddr    string
	Seeds            []string
	ProbeInterval    time.Duration
	ProbeTimeout     time.Duration
	GossipInterval   time.Duration
	PushPullInterval time.Duration
	SuspicionMult    int
}

func loadMemberlistConfig() (MemberlistConfig, error) {
	bindAddr := os.Getenv("MEMBERLIST_BIND_ADDR")
	bindPort := os.Getenv("MEMBERLIST_BIND_PORT")
	seeds := os.Getenv("MEMBERLIST_SEEDS")

	if bindAddr == "" {
		return MemberlistConfig{}, fmt.Errorf("MEMBERLIST_BIND_ADDR is required")
	}
	if bindPort == "" {
		return MemberlistConfig{}, fmt.Errorf("MEMBERLIST_BIND_PORT is required")
	}
	if seeds == "" {
		return MemberlistConfig{}, fmt.Errorf("MEMBERLIST_SEEDS is required")
	}

	port, err := strconv.Atoi(bindPort)
	if err != nil {
		return MemberlistConfig{}, fmt.Errorf("invalid MEMBERLIST_BIND_PORT: %w", err)
	}

	return MemberlistConfig{
		BindAddr:         bindAddr,
		BindPort:         port,
		AdvertiseAddr:    getEnvOrDefault("MEMBERLIST_ADVERTISE_ADDR", bindAddr),
		Seeds:            parseSeeds(seeds),
		ProbeInterval:    getDurationEnv("MEMBERLIST_PROBE_INTERVAL", 1*time.Second),
		ProbeTimeout:     getDurationEnv("MEMBERLIST_PROBE_TIMEOUT", 500*time.Millisecond),
		GossipInterval:   getDurationEnv("MEMBERLIST_GOSSIP_INTERVAL", 200*time.Millisecond),
		PushPullInterval: getDurationEnv("MEMBERLIST_PUSH_PULL_INTERVAL", 5*time.Second),
		SuspicionMult:    getIntEnv("MEMBERLIST_SUSPICION_MULT", 4),
	}, nil
}

func parseSeeds(v string) []string {
	var seeds []string
	for s := range strings.SplitSeq(v, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			seeds = append(seeds, s)
		}
	}

	return seeds
}
