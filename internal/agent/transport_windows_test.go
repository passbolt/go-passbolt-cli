package agent

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func testEndpoint(t *testing.T) string {
	return fmt.Sprintf(`\\.\pipe\passbolt-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
}
