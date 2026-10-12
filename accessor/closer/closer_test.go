package closer

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCloser_Close(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	stuck := make(chan struct{})
	defer close(stuck)

	var calls []string
	record := func(name string, err error) func() error {
		return func() error {
			calls = append(calls, name)
			return err
		}
	}
	panics := func() error {
		panic(assert.AnError)
	}
	blocks := func() error {
		<-stuck
		return nil
	}

	type testCase struct {
		name      string
		timeout   time.Duration
		closures  []func() error
		wantCalls []string
		wantErr   error
	}

	tests := []testCase{
		{
			name:      "closes in reverse order",
			timeout:   time.Second,
			closures:  []func() error{record("first", nil), record("second", nil), record("third", nil)},
			wantCalls: []string{"third", "second", "first"},
			wantErr:   nil,
		},
		{
			name:      "error does not skip other closures",
			timeout:   time.Second,
			closures:  []func() error{record("first", nil), record("second", assert.AnError), record("third", nil)},
			wantCalls: []string{"third", "second", "first"},
			wantErr:   assert.AnError,
		},
		{
			name:      "panic does not skip other closures",
			timeout:   time.Second,
			closures:  []func() error{record("first", nil), panics},
			wantCalls: []string{"first"},
			wantErr:   assert.AnError,
		},
		{
			name:      "context expired before close",
			timeout:   0,
			closures:  []func() error{record("first", nil)},
			wantCalls: nil,
			wantErr:   context.DeadlineExceeded,
		},
		{
			name:      "stuck closure is bounded by context",
			timeout:   50 * time.Millisecond,
			closures:  []func() error{record("first", nil), blocks},
			wantCalls: nil,
			wantErr:   context.DeadlineExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls = nil
			c := New(logger, tc.closures...)

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			err := c.Close(ctx)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantCalls, calls)
		})
	}
}
