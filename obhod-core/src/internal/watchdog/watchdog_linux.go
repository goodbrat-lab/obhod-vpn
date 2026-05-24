//go:build linux

package watchdog

import (
	"net"
	"syscall"
	"time"
)

func dialWithMark(network, address string, mark int, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	if mark != 0 {
		d.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, mark)
			})
		}
	}
	return d.Dial(network, address)
}
