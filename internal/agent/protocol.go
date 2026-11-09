package agent

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const ProtocolVersion = 1
const MaxFrame = 8 << 20

var ErrProtocol = errors.New("incompatible agent protocol; stop the old agent and retry")

// Request travels only over owner-restricted local IPC, never a network listener.
type Request struct {
	Terminal    string
	PID         int
	Version     int
	Action      string
	Args        []string
	Settings    map[string]any
	Directory   string
	Environment []string
	Dialog      string
	Pinentry    string
}

func (r Request) Validate() error {
	if r.Version != ProtocolVersion {
		return ErrProtocol
	}
	switch r.Action {
	case "run", "status", "lock", "stop":
		return nil
	}
	return fmt.Errorf("unknown agent action")
}

// Message multiplexes command output and interactive input without consuming stdin
// before the command actually asks for it.
type Message struct {
	Kind        string
	Data        []byte   `json:",omitempty"`
	Error       string   `json:",omitempty"`
	Status      []Status `json:",omitempty"`
	Environment []string `json:",omitempty"`
	Args        []string `json:",omitempty"`
}

func ReadFrame(r io.Reader, value any) error {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > MaxFrame {
		return fmt.Errorf("invalid agent frame size")
	}
	data := make([]byte, size)
	defer clear(data)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
func WriteFrame(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	defer clear(data)
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > MaxFrame {
		return fmt.Errorf("agent frame too large")
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
