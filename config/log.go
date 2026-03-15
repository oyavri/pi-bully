package config

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogConfig struct {
	Level  string // debug | info | warn | error
	Format string // text | json
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

func loadLogConfig() LogConfig {
	return LogConfig{
		Level:  getEnvOrDefault("LOG_LEVEL", "info"),
		Format: getEnvOrDefault("LOG_FORMAT", "text"),
	}
}
