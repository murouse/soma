// Package closer предоставляет инструменты для корректного освобождения ресурсов.
package closer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// Closer управляет списком функций завершения работы.
type Closer struct {
	mu       sync.Mutex
	closures []func() error

	logger *slog.Logger
}

// New создает новый экземпляр Closer.
func New(logger *slog.Logger, closures ...func() error) *Closer {
	return &Closer{closures: closures, logger: logger}
}

// Add регистрирует функцию завершения работы.
func (c *Closer) Add(funcs ...func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closures = append(c.closures, funcs...)
}

// Close выполняет все зарегистрированные функции последовательно в обратном порядке (LIFO).
// Перед каждым шагом проверяется состояние контекста.
func (c *Closer) Close(ctx context.Context) error {
	c.mu.Lock()
	closures := c.closures
	c.closures = nil
	c.mu.Unlock()

	errs := make([]error, 0, len(closures))

	for i := len(closures) - 1; i >= 0; i-- {
		c.logger.DebugContext(ctx, "closing resource", slog.Int("num", i))

		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("closer stopped by context: %w", err))
			break
		}

		errs = append(errs, func(f func() error) (fErr error) {
			defer func() {
				if r := recover(); r != nil {
					fErr = fmt.Errorf("panic: %v", r)
				}
			}()
			return f()
		}(closures[i]))
	}

	return errors.Join(errs...)
}
