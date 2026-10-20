package grpcconnect

import (
	"context"
	"io"
	"sync"

	"connectrpc.com/connect/v2"
)

// serializedTransport обходит гонку в vanguard v0.5.0: forward закрывает upstream-поток (defer Close),
// не дождавшись своей горутины отправки, и та зовет Send после Close — хотя контракт ClientStream
// это запрещает. Close стартует запрос и клонирует его заголовки, а Send в это время их пишет:
// при обрыве запроса клиентом процесс падает с «concurrent map iteration and map write».
// Здесь Send, CloseSend и Close не пересекаются, а после Close поток больше не трогается.
type serializedTransport struct {
	connect.Transport
}

func (t serializedTransport) NewClientStream(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
	stream, err := t.Transport.NewClientStream(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &serializedStream{ClientStream: stream}, nil
}

type serializedStream struct {
	connect.ClientStream

	mu     sync.Mutex
	closed bool
}

func (s *serializedStream) SendHeaders() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.EOF
	}
	return s.ClientStream.SendHeaders()
}

// Send после Close — io.EOF, как у закрытого потока: vanguard принимает его как «upstream закрылся»
func (s *serializedStream) Send(msg any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.EOF
	}
	return s.ClientStream.Send(msg)
}

func (s *serializedStream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	return s.ClientStream.CloseSend()
}

// Close ждет отправку, начатую до него; Receive не блокирует — Close должен уметь его прервать
func (s *serializedStream) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return s.ClientStream.Close()
}
