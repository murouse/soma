package profiler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/murouse/soma/component/httpserver"
)

// Server представляет собой HTTP-сервер для профилирования через pprof.
//
// Импорт net/http/pprof в init() регистрирует /debug/pprof в http.DefaultServeMux.
// Поэтому httpserver отказывается обслуживать DefaultServeMux (ErrDefaultServeMux),
// и в собственном коде отдавать его наружу тоже нельзя.
type Server struct {
	cfg        *Config
	httpServer *httpserver.Server
}

func New(cfg *Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Name() string {
	return "profiler"
}

func (s *Server) Prepare(ctx context.Context) error {
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

	var err error
	s.httpServer, err = httpserver.NewDefaultWith(s.cfg.Port, mux,
		httpserver.WithName(s.Name()),
		httpserver.WithLogger(s.cfg.Logger),
		httpserver.WithHost(s.cfg.Host),
		httpserver.WithShutdownTimeout(s.cfg.ShutdownTimeout),
	)
	if err != nil {
		return fmt.Errorf("new http server: %w", err)
	}

	if err = s.httpServer.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare http server: %w", err)
	}

	return nil
}

// Addr возвращает фактический адрес после Prepare (полезно при порте 0).
func (s *Server) Addr() net.Addr {
	return s.httpServer.Addr()
}

func (s *Server) Run(ctx context.Context) error {
	return s.httpServer.Run(ctx)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
