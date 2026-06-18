//go:build !windows

package collector

import (
	"context"
	"net"
)

// dockerDialContext returns a dialer for the Docker Unix socket.
func dockerDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return net.Dial("unix", dockerSocketPath)
}
