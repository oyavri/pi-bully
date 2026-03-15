package config

import (
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type DatabaseConfig struct {
	DSN string
}

type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // text | json
}

type NodeConfig struct {
	ID      uint64 // Node ID
	Address string // IP address of the node
}

type Config struct {
	Database DatabaseConfig
	Log      LogConfig
	Node     NodeConfig
}

func Load() (Config, error) {
	cfg := defaults()

	// Node
	if v := os.Getenv("NODE_ID"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("invalid NODE_ID: %w", err)
		}
		cfg.Node.ID = id
	}

	if v := os.Getenv("NODE_ADDRESS"); v != "" {
		cfg.Node.Address = v
	}

	// DSN
	cfg.Database = loadDatabaseConfig()

	// Log
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}

	return cfg, cfg.validate()
}

func defaults() Config {
	return Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func (c *Config) validate() error {
	if c.Node.ID == 0 {
		return fmt.Errorf("NODE_ID is required and must be > 0")
	}

	if c.Node.Address == "" {
		return fmt.Errorf("NODE_ADDRESS is required")
	}

	if os.Getenv("POSTGRES_HOST") == "" {
        return fmt.Errorf("POSTGRES_HOST is required")
    }

    if os.Getenv("POSTGRES_PORT") == "" {
        return fmt.Errorf("POSTGRES_PORT is required")
    }

    if os.Getenv("POSTGRES_USER") == "" {
        return fmt.Errorf("POSTGRES_USER is required")
    }

    if os.Getenv("POSTGRES_PASSWORD") == "" {
        return fmt.Errorf("POSTGRES_PASSWORD is required")
    }

    if os.Getenv("POSTGRES_DB") == "" {
        return fmt.Errorf("POSTGRES_DB is required")
    }

    if os.Getenv("POSTGRES_SSL_MODE") == "" {
        return fmt.Errorf("POSTGRES_SSL_MODE is required")
    }

	return nil
}

func (c LogConfig) BuildLogger() (*zap.Logger, error) {
	var zapCfg zap.Config

	switch c.Format {
	case "json":
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	case "text":
		zapCfg = zap.NewDevelopmentConfig()
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(c.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", c.Level, err)
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	return zapCfg.Build()
}

func loadDatabaseConfig() DatabaseConfig {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	sslMode := os.Getenv("POSTGRES_SSL_MODE")

	return DatabaseConfig{
		DSN: fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, password, host, port, dbName, sslMode
		)
	}
}
