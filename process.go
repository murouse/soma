package soma

import "context"

// Process выполняется в течение жизни программы
type Process interface {
	PreRun(ctx context.Context) error // выполняется перед запуском Run
	Run(ctx context.Context) error    // блокирующий
	Shutdown(ctx context.Context) error
}
