package profiler

import "time"

func Default() *Config {
	return &Config{
		Port:              1491,
		ShutdownTimeout:   time.Second,
		ReadHeaderTimeout: time.Minute,
	}
}
