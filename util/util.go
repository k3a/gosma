package util

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// EnsureAddressWithPort ensures that the address contains a port.
// Appends a defaultPort if it doesn't contain a port.
func EnsureAddressWithPort(address string, defaultPort int) string {
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

// SocketClient creates a socket and dials the address. Optionally it can also negotiate TLS.
// Parameter optionalServerName is used for TLS server certification
// validation and is derived from the address by default.
func SocketClient(network string, address string, timeout time.Duration, requireTLS bool, optionalServerName string) (io.ReadWriteCloser, error) {
	if optionalServerName == "" && (network == "tcp" || network == "udp") {
		h, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		optionalServerName = h
	}

	conn, err := net.DialTimeout(network, address, timeout)

	if err == nil && requireTLS {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: optionalServerName,
		})
		return tlsConn, tlsConn.Handshake()
	}

	return conn, err
}

// CompactDuration creates a duration string with maximum of maxParts parts.
// One part is a string XXXy where XXX is a number with arbitrary number
// of digits and y is a unit character (y, d, h, m, s)
func CompactDuration(duration time.Duration, maxParts int) (out string) {
	if maxParts <= 0 {
		panic("CompactDuration: maxParts must be higher or equal 1")
	}
	parts := 0

	durationsText := []string{"y", "d", "h", "m", "s"}
	durations := []time.Duration{365 * 24 * time.Hour, 24 * time.Hour, time.Hour, time.Minute, time.Second}

	for n, d := range durations {
		thisDur := duration / d
		thisDurInt := int(thisDur)
		if thisDurInt > 0 {
			duration -= time.Duration(thisDurInt) * d
			out += fmt.Sprintf("%d%s", thisDurInt, durationsText[n])
			parts++
		}

		if parts == maxParts {
			break
		}
	}

	return
}
