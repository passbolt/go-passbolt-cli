package credential

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	dockercredentials "github.com/docker/docker-credential-helpers/credentials"
)

// DockerBackend implements Docker's upstream interface. Lookups stay read-only.
type DockerBackend struct {
	Context  context.Context
	Resolver Resolver
}

var _ dockercredentials.Helper = (*DockerBackend)(nil)

func (*DockerBackend) Add(*dockercredentials.Credentials) error { return ErrReadOnly }
func (*DockerBackend) Delete(string) error                      { return ErrReadOnly }
func (b *DockerBackend) Get(serverURL string) (string, string, error) {
	key, err := DockerRegistryKey(serverURL)
	if err != nil {
		return "", "", ErrRequest
	}
	username, secret, err := b.Resolver.Docker(b.Context, key)
	if errors.Is(err, ErrNotFound) {
		return "", "", dockercredentials.NewErrCredentialsNotFound()
	}
	if err != nil {
		return "", "", err
	}
	if username == "" || strings.TrimSpace(secret) == "" || strings.ContainsAny(username+secret, "\x00") {
		return "", "", ErrResponse
	}
	return username, secret, nil
}
func (b *DockerBackend) List() (map[string]string, error) { return b.Resolver.DockerList(b.Context) }

// HandleDocker delegates wire handling to Docker without Serve's stdout error
// printing. Buffering also works around upstream Get ignoring writer failures.
func HandleDocker(helper dockercredentials.Helper, action string, input io.Reader, output io.Writer) error {
	limited := &io.LimitedReader{R: input, N: maxRequest + 1}
	var buffer bytes.Buffer
	if err := dockercredentials.HandleCommand(helper, action, limited, &buffer); err != nil {
		return err
	}
	if limited.N <= 0 {
		return ErrRequest
	}
	data := buffer.Bytes()
	defer clear(data)
	_, err := output.Write(data)
	return err
}

type captureDockerRequest struct{ server string }

func (c *captureDockerRequest) Get(server string) (string, string, error) {
	if _, err := DockerRegistryKey(server); err != nil {
		return "", "", ErrRequest
	}
	c.server = server
	return "", "", nil
}
func (*captureDockerRequest) Add(*dockercredentials.Credentials) error { return ErrReadOnly }
func (*captureDockerRequest) Delete(string) error                      { return ErrReadOnly }
func (*captureDockerRequest) List() (map[string]string, error)         { return nil, ErrRequest }
func ReadDockerServer(input io.Reader) (string, string, error) {
	// Validate framing and size before admission; Docker owns actual parsing.
	data, err := io.ReadAll(io.LimitReader(input, 65537))
	defer clear(data)
	if err != nil || len(data) > 65536 || strings.ContainsAny(strings.TrimRight(string(data), "\r\n"), "\r\n") {
		return "", "", ErrRequest
	}
	capture := &captureDockerRequest{}
	if err := dockercredentials.HandleCommand(capture, dockercredentials.ActionGet, bytes.NewReader(data), io.Discard); err != nil {
		return "", "", ErrRequest
	}
	key, err := DockerRegistryKey(capture.server)
	return capture.server, key, err
}
