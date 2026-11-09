package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"os/exec"
	"syscall"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func logonSID() (string, error) {
	return tokenLogonSID(windows.GetCurrentProcessToken())
}
func tokenLogonSID(token windows.Token) (string, error) {
	groups, err := token.GetTokenGroups()
	if err != nil {
		return "", err
	}
	for _, group := range groups.AllGroups() {
		if group.Attributes&windows.SE_GROUP_LOGON_ID == windows.SE_GROUP_LOGON_ID {
			return group.Sid.String(), nil
		}
	}
	return "", fmt.Errorf("cannot determine Windows logon identity")
}
func Endpoint() (string, error) {
	sid, err := logonSID()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`\\.\pipe\passbolt-agent-%x`, sha256.Sum256([]byte(sid))), nil
}
func Listen(endpoint string) (net.Listener, error) {
	sid, err := logonSID()
	if err != nil {
		return nil, err
	}
	return winio.ListenPipe(endpoint, &winio.PipeConfig{SecurityDescriptor: "D:P(A;;GA;;;" + sid + ")"})
}
func Dial(ctx context.Context, endpoint string) (net.Conn, error) {
	conn, err := winio.DialPipeContext(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	// A restrictive listener DACL alone cannot prevent a different user from
	// squatting on the pipe name. Authenticate the connected server before sending
	// any settings, key material, or environment values.
	if err := verifyPipeServer(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}
func Detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS, HideWindow: true}
}

func verifyPipeServer(conn net.Conn) error {
	descriptor, ok := conn.(interface{ Fd() uintptr })
	if !ok {
		return fmt.Errorf("cannot verify agent pipe server")
	}
	var pid uint32
	if err := windows.GetNamedPipeServerProcessId(windows.Handle(descriptor.Fd()), &pid); err != nil {
		return err
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err = windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer func() { _ = token.Close() }()
	expected, err := logonSID()
	if err != nil {
		return err
	}
	actual, err := tokenLogonSID(token)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("agent pipe server belongs to a different Windows logon")
	}
	return nil
}
