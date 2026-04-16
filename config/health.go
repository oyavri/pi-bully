package config

type HealthConfig struct {
	Port string
}

func loadHealthConfig() HealthConfig {
	return HealthConfig{
		Port: getEnvOrDefault("HEALTH_PORT", "8080"),
	}
}
