package agent

import (
	"context"
	"testing"
	"time"
)

func TestLocalTransportRoundTripAndSingleListener(t *testing.T) {
	endpoint := testEndpoint(t)
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if second, err := Listen(endpoint); err == nil {
		second.Close()
		t.Fatal("second server replaced live listener")
	}
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		var req Request
		if err = ReadFrame(conn, &req); err == nil {
			err = WriteFrame(conn, Message{Kind: req.Action})
		}
		done <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := WriteFrame(conn, Request{Version: ProtocolVersion, Action: "status"}); err != nil {
		t.Fatal(err)
	}
	var msg Message
	if err := ReadFrame(conn, &msg); err != nil || msg.Kind != "status" {
		t.Fatalf("round trip: %+v %v", msg, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
