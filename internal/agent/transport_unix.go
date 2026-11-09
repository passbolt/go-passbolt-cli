//go:build !windows

package agent

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func Endpoint() (string, error) {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("passbolt-agent-%d", os.Getuid()))
	if err := os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
		return "", err
	}
	if err := privateDirectory(dir); err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent.sock"), nil
}
func privateDirectory(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode().Perm()&0077 != 0 || stat.Uid != uint32(os.Getuid()) {
		return fmt.Errorf("agent directory must be owned by you with mode 0700: %s", dir)
	}
	return nil
}

type localListener struct {
	net.Listener
	lock     *os.File
	endpoint string
}

func (l *localListener) Close() error {
	err := l.Listener.Close()
	_ = os.Remove(l.endpoint)
	_ = unix.Flock(int(l.lock.Fd()), unix.LOCK_UN)
	_ = l.lock.Close()
	return err
}
func Listen(endpoint string) (net.Listener, error) {
	if err := privateDirectory(filepath.Dir(endpoint)); err != nil {
		return nil, err
	}
	fd, err := unix.Open(endpoint+".lock", unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	lock := os.NewFile(uintptr(fd), endpoint+".lock")
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("agent already running: %w", err)
	}
	// Only the lock owner may remove a stale socket.
	_ = os.Remove(endpoint)
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		_ = lock.Close()
		return nil, err
	}
	if err = os.Chmod(endpoint, 0600); err != nil {
		_ = listener.Close()
		_ = lock.Close()
		return nil, err
	}
	return &localListener{listener, lock, endpoint}, nil
}
func Dial(ctx context.Context, endpoint string) (net.Conn, error) {
	if err := privateDirectory(filepath.Dir(endpoint)); err != nil {
		return nil, err
	}
	info, err := os.Lstat(endpoint)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0077 != 0 || stat.Uid != uint32(os.Getuid()) {
		return nil, fmt.Errorf("unsafe agent socket")
	}
	return (&net.Dialer{}).DialContext(ctx, "unix", endpoint)
}
func Detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
