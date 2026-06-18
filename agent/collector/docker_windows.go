//go:build windows

package collector

import (
	"context"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

var dockerEndpoint = "127.0.0.1:2375"

func init() {
	if v := os.Getenv("DOCKER_HOST"); v != "" {
		dockerEndpoint = v
	}
}

// dockerDialContext connects to the Docker daemon on Windows.
// Priority: named pipe (Docker Desktop default) → TCP (DOCKER_HOST or 127.0.0.1:2375).
func dockerDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Named pipe is the default for Docker Desktop on Windows
	if conn, err := winio.DialPipeContext(ctx, `\\.\pipe\docker_engine`); err == nil {
		return conn, nil
	}
	// Fall back to TCP (user set DOCKER_HOST, or enabled TCP in Docker Desktop settings)
	var dialer net.Dialer
	return dialer.DialContext(ctx, "tcp", dockerEndpoint)
}
