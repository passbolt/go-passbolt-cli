package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func linuxPTY(t *testing.T) (int, string, *os.File) {
	t.Helper()
	master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { unix.Close(master) })
	if err = unix.IoctlSetPointerInt(master, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/dev/pts/%d", number)
	slave, err := os.OpenFile(path, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { slave.Close() })
	return master, path, slave
}

func TestTerminalCancellationRestoresMode(t *testing.T) {
	_, path, slave := linuxPTY(t)
	before, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	dialog := Dialog{Backend: "terminal", Terminal: path}
	if _, err = dialog.Ask(ctx, "test", "confirm", false); err == nil {
		t.Fatal("cancellation accepted")
	}
	after, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("terminal mode was not restored after cancellation")
	}
}

func TestTerminalApprovalAndSecretInput(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
		secret, deny      bool
	}{
		{name: "approve", input: "y"},
		{name: "default deny", input: "\r", deny: true},
		{name: "masked Unicode editing", input: "p💡\x7fZ\r", want: "pZ", secret: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			master, path, _ := linuxPTY(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			type reply struct {
				value string
				err   error
			}
			result := make(chan reply, 1)
			go func() {
				value, err := (Dialog{Backend: "terminal", Terminal: path}).Ask(ctx, "Passbolt", "test", test.secret)
				result <- reply{value, err}
			}()
			var output strings.Builder
			prompt := "[y]"
			if test.secret {
				prompt = "hidden"
			}
			for !strings.Contains(output.String(), prompt) {
				ready, err := unix.Poll([]unix.PollFd{{Fd: int32(master), Events: unix.POLLIN}}, 100)
				if err != nil {
					t.Fatal(err)
				}
				if ready == 0 {
					if ctx.Err() != nil {
						t.Fatal("dialog never appeared")
					}
					continue
				}
				var data [1024]byte
				n, err := unix.Read(master, data[:])
				if err != nil {
					t.Fatal(err)
				}
				output.Write(data[:n])
			}
			if _, err := unix.Write(master, []byte(test.input)); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-result:
				if test.deny {
					if !errors.Is(got.err, ErrDenied) {
						t.Fatal(got.err)
					}
				} else if got.err != nil || got.value != test.want {
					t.Fatalf("value=%q err=%v", got.value, got.err)
				}
			case <-ctx.Done():
				t.Fatal("dialog did not finish")
			}
			ready, _ := unix.Poll([]unix.PollFd{{Fd: int32(master), Events: unix.POLLIN}}, 20)
			if ready > 0 {
				var data [1024]byte
				n, _ := unix.Read(master, data[:])
				output.Write(data[:n])
			}
			if test.secret && strings.Contains(output.String(), "💡") {
				t.Fatal("secret was echoed to the terminal")
			}
		})
	}
}

func TestTerminalCredentialSelection(t *testing.T) {
	master, path, slave := linuxPTY(t)
	before, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan int, 1)
	go func() {
		at, err := (Dialog{Backend: "terminal", Terminal: path}).Select(ctx, "Credentials", "Choose", []string{"https://git.example — alice", "https://git.example — bob"})
		if err != nil {
			result <- -1
		} else {
			result <- at
		}
	}()
	var output strings.Builder
	for !strings.Contains(output.String(), "Option number") {
		n, err := unix.Poll([]unix.PollFd{{Fd: int32(master), Events: unix.POLLIN}}, 50)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			if ctx.Err() != nil {
				t.Fatal("selection did not render")
			}
			continue
		}
		var data [4096]byte
		n, err = unix.Read(master, data[:])
		if err != nil {
			t.Fatal(err)
		}
		output.Write(data[:n])
	}
	unix.Write(master, []byte("2\r"))
	select {
	case at := <-result:
		if at != 1 {
			t.Fatalf("selected %d", at)
		}
	case <-ctx.Done():
		t.Fatal("selection did not finish")
	}
	after, err := term.GetState(int(slave.Fd()))
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("selection did not restore terminal mode")
	}
}
