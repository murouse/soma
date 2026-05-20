package profiler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
)

// Server представляет собой HTTP-сервер для профилирования через pprof.
type Server struct {
	cfg *Config

	server *http.Server
}

func New(cfg *Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Prepare(_ context.Context) error {
	mux := http.NewServeMux()

	// Index responds with the pprof-formatted profile named by the request.
	// For example, "/debug/pprof/heap" serves the "heap" profile.
	// Index responds to a request for "/debug/pprof/" with an HTML page
	// listing the available profiles.
	mux.HandleFunc("/debug/pprof/", pprof.Index)

	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))

	// Block responds to a request for "/debug/pprof/block" with
	// the running block profile.
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))

	// Mutex responds to a request for "/debug/pprof/mutex" with
	// the running mutex profile.
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

	// Thread creates a thread contention profile.
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	// Allocs responds to a request for "/debug/pprof/allocs" with
	// the running allocation profile.
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	// Profile responds to a request for "/debug/pprof/profile" with
	// the pprof-formatted cpu profile.
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)

	// Symbol looks up the program counters listed in the request
	// and writes a table of the corresponding function names
	// and line numbers to w.
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Trace responds to a request for "/debug/pprof/trace" with
	// the execution trace in binary form.
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Cmdline responds to a request for "/debug/pprof/cmdline" with
	// the command line of the current program.
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)

	s.server = &http.Server{
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		Addr:              fmt.Sprintf(":%d", s.cfg.Port),
		Handler:           mux,
	}

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
