package grpcinterceptor

import "log/slog"

type Manager struct {
	logger              *slog.Logger
	loggingMetadataKeys []string
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger: logger,
		loggingMetadataKeys: []string{
			"user-agent",
		},
	}
}
