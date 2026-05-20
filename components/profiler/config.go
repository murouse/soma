package profiler

import (
	"time"
)

type Config struct {
	Port              int
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
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
