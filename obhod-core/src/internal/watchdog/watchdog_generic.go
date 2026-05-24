//go:build !linux

package watchdog

import (
	"net"
	"time"
)

func dialWithMark(network, address string, mark int, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial(network, address)
}
