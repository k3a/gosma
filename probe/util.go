package probe

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

func ensureAddressWithPort(address string, defaultPort int) string {
	var err error

	// no colon means no port
	if !strings.Contains(address, ":") {
		goto addDefaultPort
	}

	// may be an ipv6 address without a port
	_, _, err = net.SplitHostPort(address)
	if err != nil && strings.Contains(err.Error(), "missing port") {
		goto addDefaultPort
	}

	// is ok, pass through
	return address

addDefaultPort:
	return address + ":" + fmt.Sprint(defaultPort)
}

func socketClient(network string, address string, timeout time.Duration, requireTLS bool) (io.ReadWriteCloser, error) {
	h, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout(network, address, timeout)

	if err == nil && requireTLS {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: h,
		})
		return tlsConn, tlsConn.Handshake()
	}

	return conn, err
}
