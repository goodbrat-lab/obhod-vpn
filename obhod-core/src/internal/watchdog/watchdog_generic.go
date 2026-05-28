//go:build !linux

package watchdog

import (
	"context"
	"net"
	"time"
)

func dialWithMark(ctx context.Context, network, address string, mark int, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, network, address)
}
