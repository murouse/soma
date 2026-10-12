package soma

import (
	"fmt"
	"time"
)

// Default возвращает конфигурацию без компонентов: каждый компонент (scheduler, gRPC-сервер,
// gateway, профайлер) включается только явно своей опцией With*.
func Default() *EntrypointConfig {
	return &EntrypointConfig{
		prepareTimeout:  5 * time.Second,
		shutdownTimeout: 5 * time.Second,
		closeTimeout:    5 * time.Second,
	}
}

func DefaultWith(opts ...EntrypointOption) (*EntrypointConfig, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return cfg, nil
}
