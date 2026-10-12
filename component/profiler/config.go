package profiler

import (
	"fmt"
	"log/slog"
	"time"
)

type Config struct {
	Host            string
	Port            int
	ShutdownTimeout time.Duration
	Logger          *slog.Logger // nil — slog.Default()
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

func WithPort(port int) Option {
	return func(c *Config) error {
		c.Port = port
		return nil
	}
}

// WithHost задает интерфейс, на котором слушает профайлер. Пустая строка — все интерфейсы:
// pprof отдает дампы памяти и командную строку процесса, поэтому открывать его наружу стоит осознанно.
func WithHost(host string) Option {
	return func(c *Config) error {
		c.Host = host
		return nil
	}
}

func WithShutdownTimeout(shutdownTimeout time.Duration) Option {
	return func(c *Config) error {
		c.ShutdownTimeout = shutdownTimeout
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}
