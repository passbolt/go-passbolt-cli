package agent

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Handler interface {
	Run(context.Context, Request, io.Reader, io.Writer, io.Writer) (Message, error)
	Control(context.Context, Request) (Message, error)
	Expire(context.Context)
	Close(context.Context)
}

// Serve serializes commands and all session lifecycle operations. Connections are
// bounded; slow/abandoned clients cannot retain a session or refresh its lease.
func Serve(ctx context.Context, listener net.Listener, handler Handler) error {
	defer func() { _ = listener.Close() }()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		handler.Close(cleanup)
	}()
	type pending struct {
		conn net.Conn
		req  Request
	}
	requests := make(chan pending)
	failures := make(chan error, 1)
	slots := make(chan struct{}, 16)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				failures <- err
				return
			}
			select {
			case slots <- struct{}{}:
			default:
				_ = conn.Close()
				continue
			}
			go func() {
				defer func() { <-slots }()
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				var req Request
				err := ReadFrame(conn, &req)
				_ = conn.SetReadDeadline(time.Time{})
				if err == nil {
					err = req.Validate()
				}
				if err != nil {
					_ = WriteFrame(conn, Message{Kind: "done", Error: err.Error()})
					_ = conn.Close()
					return
				}
				if req.Action == "browser-pending" || req.Action == "browser-submit" || req.Action == "browser-wait" || req.Action == "browser-cancel" {
					waitCtx, waitCancel := context.WithTimeout(ctx, 35*time.Second)
					defer waitCancel()
					var result Message
					if browser, ok := handler.(interface {
						Browser(context.Context, Request) (Message, error)
					}); ok {
						result, err = browser.Browser(waitCtx, req)
					} else {
						err = fmt.Errorf("browser authentication is unavailable")
					}
					if err != nil {
						result = Message{Kind: "done", Error: err.Error()}
					}
					_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					_ = WriteFrame(conn, result)
					_ = conn.Close()
					return
				}
				select {
				case requests <- pending{conn, req}:
				case <-ctx.Done():
					_ = conn.Close()
				}
			}()
		}
	}()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-failures:
			return err
		case <-ticker.C:
			handler.Expire(ctx)
		case request := <-requests:
			handler.Expire(ctx)
			handle(ctx, request.conn, request.req, handler)
			if request.req.Action == "stop" {
				return nil
			}
		}
	}
}

type wire struct {
	mu   sync.Mutex
	conn net.Conn
}

func (w *wire) send(msg Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := WriteFrame(w.conn, msg)
	_ = w.conn.SetWriteDeadline(time.Time{})
	return err
}

type output struct {
	wire *wire
	kind string
}

func (w output) Write(data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		n := min(len(data), 32*1024)
		if err := w.wire.send(Message{Kind: w.kind, Data: data[:n]}); err != nil {
			return total, err
		}
		total += n
		data = data[n:]
	}
	return total, nil
}

type input struct {
	wire    *wire
	ctx     context.Context
	replies <-chan Message
	pending []byte
}

func (r *input) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		if err := r.wire.send(Message{Kind: "input"}); err != nil {
			return 0, err
		}
		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case reply := <-r.replies:
			if reply.Kind != "input" {
				return 0, fmt.Errorf("unexpected agent input")
			}
			if len(reply.Data) == 0 {
				return 0, io.EOF
			}
			r.pending = reply.Data
		}
	}
	n := copy(data, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}
func handle(parent context.Context, conn net.Conn, req Request, handler Handler) {
	defer func() { _ = conn.Close() }()
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	replies := make(chan Message, 1)
	go func() {
		defer cancel()
		for {
			var msg Message
			if ReadFrame(conn, &msg) != nil {
				return
			}
			if msg.Kind != "input" {
				return
			}
			select {
			case replies <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	w := &wire{conn: conn}
	var result Message
	var err error
	if req.Action == "run" {
		result, err = handler.Run(ctx, req, &input{wire: w, ctx: ctx, replies: replies}, output{w, "stdout"}, output{w, "stderr"})
	} else {
		result, err = handler.Control(ctx, req)
	}
	if result.Kind == "" {
		result.Kind = "done"
	}
	if err != nil {
		result.Error = err.Error()
	}
	_ = w.send(result)
}

func Connect(ctx context.Context, endpoint string, start bool) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := Dial(ctx, endpoint)
	if err == nil || !start {
		return conn, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(executable, "__agent-serve")
	// Credentials from config/environment must not leak into daemon launch state.
	for _, name := range []string{"PATH", "HOME", "USER", "USERPROFILE", "SYSTEMROOT", "WINDIR", "LOCALAPPDATA", "APPDATA", "TMP", "TEMP", "TMPDIR", "LANG", "LC_ALL"} {
		if value, ok := os.LookupEnv(name); ok {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	Detach(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() { _ = cmd.Wait() }()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return nil, fmt.Errorf("agent did not become ready: %w", err)
		case <-ticker.C:
			conn, err = Dial(ctx, endpoint)
			if err == nil {
				return conn, nil
			}
		}
	}
}

// Exchange never retries an admitted command: a broken connection may follow a
// successful write. Input is read only after an explicit request from the agent.
func Exchange(ctx context.Context, conn net.Conn, req Request, in io.Reader, out, errOut io.Writer) (Message, error) {
	defer func() { _ = conn.Close() }()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	if err := WriteFrame(conn, req); err != nil {
		return Message{}, err
	}
	for {
		var msg Message
		if err := ReadFrame(conn, &msg); err != nil {
			return Message{}, fmt.Errorf("agent connection lost; command was not retried: %w", err)
		}
		switch msg.Kind {
		case "stdout":
			if _, err := out.Write(msg.Data); err != nil {
				return Message{}, err
			}
		case "stderr":
			if _, err := errOut.Write(msg.Data); err != nil {
				return Message{}, err
			}
		case "input":
			data := make([]byte, 32*1024)
			n, err := in.Read(data)
			if err != nil && err != io.EOF {
				return Message{}, err
			}
			if err := WriteFrame(conn, Message{Kind: "input", Data: data[:n]}); err != nil {
				return Message{}, err
			}
		case "done", "exec":
			if msg.Error != "" {
				return msg, fmt.Errorf("%s", msg.Error)
			}
			return msg, nil
		default:
			return Message{}, fmt.Errorf("unexpected agent response")
		}
	}
}
