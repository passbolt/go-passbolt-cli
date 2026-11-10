package agent

import (
	"context"
	"io"
	"testing"
	"time"
)

type browserHandler struct {
	started chan struct{}
	release chan struct{}
}

func (h *browserHandler) Run(ctx context.Context, _ Request, _ io.Reader, _ io.Writer, _ io.Writer) (Message, error) {
	close(h.started)
	select {
	case <-h.release:
		return Message{}, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}
func (h *browserHandler) Browser(_ context.Context, _ Request) (Message, error) {
	close(h.release)
	return Message{Kind: "browser"}, nil
}
func (*browserHandler) Control(context.Context, Request) (Message, error) { return Message{}, nil }
func (*browserHandler) Expire(context.Context)                            {}
func (*browserHandler) Close(context.Context)                             {}
func TestBrowserMessagesDoNotDeadlockWaitingCommand(t *testing.T) {
	endpoint := testEndpoint(t)
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	h := &browserHandler{started: make(chan struct{}), release: make(chan struct{})}
	go Serve(ctx, listener, h)
	conn, err := Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Exchange(ctx, conn, Request{Version: ProtocolVersion, Action: "run"}, nil, io.Discard, io.Discard)
		done <- err
	}()
	<-h.started
	browserConn, err := Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer browserConn.Close()
	if err = WriteFrame(browserConn, Request{Version: ProtocolVersion, Action: "browser-pending"}); err != nil {
		t.Fatal(err)
	}
	browserConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var msg Message
	if err = ReadFrame(browserConn, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Kind != "browser" {
		t.Fatalf("unexpected response %#v", msg)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}
