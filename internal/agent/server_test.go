package agent

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

type echoHandler struct{ closed chan struct{} }

func (h *echoHandler) Run(ctx context.Context, r Request, in io.Reader, out, errOut io.Writer) (Message, error) {
	if r.Directory == "wait" {
		<-ctx.Done()
		return Message{}, ctx.Err()
	}
	out.Write([]byte("output"))
	errOut.Write([]byte("error output"))
	data := make([]byte, 4)
	n, err := in.Read(data)
	if err != nil {
		return Message{}, err
	}
	out.Write(data[:n])
	return Message{}, nil
}
func (*echoHandler) Control(context.Context, Request) (Message, error) { return Message{}, nil }
func (*echoHandler) Expire(context.Context)                            {}
func (h *echoHandler) Close(context.Context)                           { close(h.closed) }
func TestExchangeStreamsAndCancellation(t *testing.T) {
	endpoint := testEndpoint(t)
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := &echoHandler{closed: make(chan struct{})}
	go Serve(ctx, listener, handler)
	conn, err := Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	_, err = Exchange(ctx, conn, Request{Version: ProtocolVersion, Action: "run"}, bytes.NewBufferString("data"), &out, &stderr)
	if err != nil || out.String() != "outputdata" || stderr.String() != "error output" {
		t.Fatalf("%q %q %v", out.String(), stderr.String(), err)
	}
	conn, err = Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	requestCtx, requestCancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer requestCancel()
	if _, err = Exchange(requestCtx, conn, Request{Version: ProtocolVersion, Action: "run", Directory: "wait"}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("ignored cancellation")
	}
	cancel()
	select {
	case <-handler.closed:
	case <-time.After(time.Second):
		t.Fatal("server did not clean up")
	}
}
