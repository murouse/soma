// Package closer предоставляет инструменты для корректного освобождения ресурсов.
package closer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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
// Ошибка или паника одной функции не мешает выполнить остальные.
//
// Функции закрытия не принимают контекст, поэтому прервать зависшую функцию нельзя:
// по истечении ctx Close перестает ее ждать и возвращает ошибку, а оставшиеся функции не запускаются.
func (c *Closer) Close(ctx context.Context) error {
	c.mu.Lock()
	closures := c.closures
	c.closures = nil
	c.mu.Unlock()

	// Буфер 1: если ctx истечет раньше, горутина не зависнет на отправке результата.
	done := make(chan error, 1)
	go func() { done <- c.closeAll(ctx, closures) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("closer stopped by context: %w", ctx.Err())
	}
}

func (c *Closer) closeAll(ctx context.Context, closures []func() error) error {
	errs := make([]error, 0, len(closures))

	for i, closure := range slices.Backward(closures) {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("closer stopped by context: %w", err))
			break
		}

		c.logger.DebugContext(ctx, "closing resource", slog.Int("num", i))

		errs = append(errs, func(f func() error) (fErr error) {
			defer func() {
				if r := recover(); r != nil {
					// Паника с ошибкой оборачивается через %w, чтобы работал errors.Is.
					if err, ok := r.(error); ok {
						fErr = fmt.Errorf("panic: %w", err)
					} else {
						fErr = fmt.Errorf("panic: %v", r)
					}
				}
			}()
			return f()
		}(closure))
	}

	return errors.Join(errs...)
}
