package profiler

import (
	"fmt"
	"log/slog"
	"time"
)

type Config struct {
	Port              int
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	Logger            *slog.Logger
}

func (c *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

type Option func(*Config) error

func WithPort(cfg *Config) Option {
	return func(c *Config) error {
		c.Port = cfg.Port
		return nil
	}
}

func WithShutdownTimeout(shutdownTimeout time.Duration) Option {
	return func(c *Config) error {
		c.ShutdownTimeout = shutdownTimeout
		return nil
	}
}

func WithReadHeaderTimeout(readHeaderTimeout time.Duration) Option {
	return func(c *Config) error {
		c.ReadHeaderTimeout = readHeaderTimeout
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}
