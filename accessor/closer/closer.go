// Package closer предоставляет инструменты для корректного освобождения ресурсов.
package closer

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Closer управляет списком функций завершения работы.
type Closer struct {
	mu       sync.Mutex
	closures []func() error
}

// New создает новый экземпляр Closer.
func New(closures ...func() error) *Closer {
	return &Closer{closures: closures}
}

// Add регистрирует функцию завершения работы.
func (c *Closer) Add(funcs ...func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closures = append(c.closures, funcs...)
}

// Close выполняет все зарегистрированные функции в обратном порядке (LIFO).
func (c *Closer) Close(ctx context.Context) error {
	c.mu.Lock()
	closures := c.closures
	c.closures = nil
	c.mu.Unlock()

	var errs []error
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := len(closures) - 1; i >= 0; i-- {
		wg.Add(1)
		go func(f func() error) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("panic: %v", r))
					mu.Unlock()
				}
			}()

			if err := f(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(closures[i])
	}

	wg.Wait()
	return errors.Join(errs...)
}
