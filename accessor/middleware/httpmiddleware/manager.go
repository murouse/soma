package httpmiddleware

import "log/slog"

type Manager struct {
	logger            *slog.Logger
	loggingHeaderKeys []string
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger: logger,
		loggingHeaderKeys: []string{
			"User-Agent",
		},
	}
}
