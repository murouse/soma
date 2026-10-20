package grpcconnect

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/murouse/soma/component/grpcserver"
	"github.com/stretchr/testify/require"
)

// Клиент обрывает запросы (уход со страницы, отмена fetch): шлюз не должен падать.
// Гонку на заголовках запроса ловит -race; без него она роняет процесс через раз.
func TestServer_CanceledRequests(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	grpcCfg, err := grpcserver.DefaultWith(
		grpcserver.WithPort(0),
		grpcserver.WithServices(echo{}),
		grpcserver.WithLogger(logger),
	)
	require.NoError(t, err)
	grpcSrv := grpcserver.New(grpcCfg)
	require.NoError(t, grpcSrv.Prepare(ctx))
	go func() { _ = grpcSrv.Run(ctx) }()
	defer func() { _ = grpcSrv.Shutdown(ctx) }()

	cfg, err := DefaultWith(WithPort(0), WithLogger(logger))
	require.NoError(t, err)
	grpcTarget := net.JoinHostPort("localhost", strconv.Itoa(grpcSrv.Addr().(*net.TCPAddr).Port))
	s := New(cfg, grpcSrv, grpcTarget)
	require.NoError(t, s.Prepare(ctx))
	go func() { _ = s.Run(ctx) }()
	defer func() { _ = s.Shutdown(ctx) }()

	url := "http://" + s.Addr().String() + "/" + echoService + "/Echo"
	var wg sync.WaitGroup
	for range 2000 {
		wg.Go(func() {
			reqCtx, cancel := context.WithTimeout(ctx, time.Duration(rand.IntN(2000))*time.Microsecond)
			defer cancel()
			req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, strings.NewReader(`"hello"`))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		})
	}
	wg.Wait()
}
