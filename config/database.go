package config

import (
	"fmt"
	"os"
)

type DatabaseConfig struct {
	DSN string
}

func loadDatabaseConfig() (DatabaseConfig, error) {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	sslMode := os.Getenv("POSTGRES_SSL_MODE")

	if host == "" {
		return DatabaseConfig{}, fmt.Errorf("POSTGRES_HOST is required")
	}

	if port == "" {
		return DatabaseConfig{}, fmt.Errorf("POSTGRES_PORT is required")
	}

	if user == "" {
		return DatabaseConfig{}, fmt.Errorf("POSTGRES_USER is required")
	}

	if password == "" {
		return DatabaseConfig{}, fmt.Errorf("POSTGRES_PASSWORD is required")
	}

	if dbName == "" {
		return DatabaseConfig{}, fmt.Errorf("POSTGRES_DB is required")
	}

	if sslMode == "" {
		return DatabaseConfig{}, fmt.Errorf("POSTGRES_SSL_MODE is required")
	}

	return DatabaseConfig{
		DSN: fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, password, host, port, dbName, sslMode,
		),
	}, nil
}
