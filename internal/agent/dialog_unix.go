//go:build !windows

package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func CurrentTerminal() string {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return ""
	}
	defer func() { _ = tty.Close() }()
	cmd := exec.Command("tty")
	cmd.Stdin = tty
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
func (d Dialog) openTerminal() (*os.File, *os.File, func(), error) {
	if !strings.HasPrefix(d.Terminal, "/dev/") {
		return nil, nil, nil, fmt.Errorf("invalid terminal")
	}
	fd, err := unix.Open(d.Terminal, unix.O_RDWR|unix.O_NOCTTY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	file := os.NewFile(uintptr(fd), d.Terminal)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Getuid()) || !term.IsTerminal(fd) {
		file.Close()
		return nil, nil, nil, fmt.Errorf("terminal must belong to the requesting user")
	}
	return file, file, func() { file.Close() }, nil
}

// Poll in bounded intervals without closing the descriptor needed to restore
// terminal mode. Closing a terminal in another goroutine may not unblock Read.
func terminalBytes(ctx context.Context, file *os.File, data []byte) (int, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		events := []unix.PollFd{{Fd: int32(file.Fd()), Events: unix.POLLIN}}
		n, err := unix.Poll(events, 50)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			continue
		}
		return file.Read(data[:1])
	}
}
