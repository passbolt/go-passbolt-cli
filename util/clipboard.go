package util

import (
	"fmt"
	"time"

	"golang.design/x/clipboard"
)

var delaySeconds int = 5

// SetDelay Set the delay between entries
func SetDelay(delay int) {
	delaySeconds = delay
}

/*
CopyValueToClipboard Copy the given value to clipboard and sleeps for seconds defined with SetDelay (default: 5 seconds).

After the delay the clipboard will be emptied
*/
func CopyValueToClipboard(entryName string, value string) {
	clipboard.Write(clipboard.FmtText, []byte(value))
	fmt.Printf("Copying %s to clipboard...\n", entryName)
	time.Sleep(time.Duration(delaySeconds) * time.Second)
	clipboard.Write(clipboard.FmtText, []byte{})
}
