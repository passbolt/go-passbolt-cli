// passbolt-browser-host is launched by the browser's native messaging runtime.
package main

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd"
	"os"
)

func main() {
	if err := cmd.RunBrowserHost(os.Stdin, os.Stdout); err != nil { //nolint:forbidigo // browser native messaging host uses raw binary stdout
		os.Exit(1)
	}
}
