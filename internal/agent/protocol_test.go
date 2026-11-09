package agent

import (
	"bytes"
	"errors"
	"testing"
)

func TestProtocolBounds(t *testing.T) {
	for _, data := range [][]byte{{0, 0, 0, 0}, {255, 255, 255, 255}, {0, 0, 0, 9, '{'}} {
		var r Request
		if err := ReadFrame(bytes.NewReader(data), &r); err == nil {
			t.Fatal("accepted invalid frame")
		}
	}
	if err := (Request{Version: ProtocolVersion + 1}).Validate(); !errors.Is(err, ErrProtocol) {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := Request{Version: ProtocolVersion, Action: "status"}
	if err := WriteFrame(&buf, in); err != nil {
		t.Fatal(err)
	}
	var out Request
	if err := ReadFrame(&buf, &out); err != nil || out.Action != "status" {
		t.Fatalf("round trip: %v", err)
	}
}
