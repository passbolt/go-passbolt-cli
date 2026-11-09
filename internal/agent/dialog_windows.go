package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

func CurrentTerminal() string { return "console" }
func (d Dialog) openTerminal() (*os.File, *os.File, func(), error) {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	attached, _, _ := kernel.NewProc("AttachConsole").Call(uintptr(uint32(d.PID)))
	if attached == 0 {
		return nil, nil, nil, fmt.Errorf("cannot attach requesting console")
	}
	free := func() { kernel.NewProc("FreeConsole").Call() }
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		free()
		return nil, nil, nil, err
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		_ = in.Close()
		free()
		return nil, nil, nil, err
	}
	return in, out, func() { _ = in.Close(); _ = out.Close(); free() }, nil
}
func (d Dialog) desktop(ctx context.Context, title, prompt string, secret bool) (string, error) {
	data, _ := json.Marshal(map[string]any{"title": title, "prompt": prompt, "secret": secret})
	const script = `Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing;
 $r=[Console]::In.ReadToEnd() | ConvertFrom-Json;
 if (!$r.secret) { $result=[System.Windows.Forms.MessageBox]::Show($r.prompt,$r.title,'YesNo','Question','Button2'); if ($result -ne 'Yes') {exit 1}; exit 0 };
 $f=New-Object System.Windows.Forms.Form; $f.Text=$r.title; $f.Size=New-Object System.Drawing.Size(520,210); $f.StartPosition='CenterScreen';
 $l=New-Object System.Windows.Forms.Label; $l.Text=$r.prompt; $l.SetBounds(15,15,470,50); $f.Controls.Add($l);
 $t=New-Object System.Windows.Forms.TextBox; $t.UseSystemPasswordChar=$true; $t.SetBounds(15,75,470,25); $f.Controls.Add($t);
 $ok=New-Object System.Windows.Forms.Button; $ok.Text='Allow'; $ok.SetBounds(315,115,80,30); $ok.DialogResult='OK'; $f.Controls.Add($ok);
 $cancel=New-Object System.Windows.Forms.Button; $cancel.Text='Deny'; $cancel.SetBounds(405,115,80,30); $cancel.DialogResult='Cancel'; $f.Controls.Add($cancel); $f.AcceptButton=$ok; $f.CancelButton=$cancel;
 if ($f.ShowDialog() -ne 'OK') {exit 1}; [Console]::Out.Write($t.Text); $t.Clear(); $f.Dispose();`
	return d.run(ctx, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", script}, string(data))
}

// INPUT_RECORD consists of a WORD type, padding, and a 16-byte event union.
// Reading records rather than ReadFile prevents focus/resize/key-up events from
// turning a ready console handle into an uncancellable blocking character read.
type consoleInputRecord struct {
	Type       uint16
	Padding    uint16
	KeyDown    int32
	Repeat     uint16
	VirtualKey uint16
	ScanCode   uint16
	Character  uint16
	Control    uint32
}

func terminalBytes(ctx context.Context, file *os.File, data []byte) (int, error) {
	read := syscall.NewLazyDLL("kernel32.dll").NewProc("ReadConsoleInputW")
	var high uint16
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		status, err := windows.WaitForSingleObject(windows.Handle(file.Fd()), 50)
		if err != nil {
			return 0, err
		}
		if status == uint32(windows.WAIT_TIMEOUT) {
			continue
		}
		var record consoleInputRecord
		var count uint32
		result, _, callErr := read.Call(file.Fd(), uintptr(unsafe.Pointer(&record)), 1, uintptr(unsafe.Pointer(&count)))
		if result == 0 {
			return 0, callErr
		}
		if count == 0 || record.Type != windows.KEY_EVENT || record.KeyDown == 0 || record.Character == 0 {
			continue
		}
		ch := record.Character
		if ch >= 0xD800 && ch <= 0xDBFF {
			high = ch
			continue
		}
		value := rune(ch)
		if high != 0 {
			value = utf16.DecodeRune(rune(high), rune(ch))
			high = 0
		}
		return utf8.EncodeRune(data, value), nil
	}
}
