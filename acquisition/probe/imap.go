package probe

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/k3a/gosma/util"
)

type IMAP struct {
	base

	Server string `yaml:"server"`
	TLS    bool   `yaml:"tls"`
}

func (p *IMAP) err(f string, args ...any) error {
	return fmt.Errorf("imap: "+f, args...)
}

func (p *IMAP) String() string {
	attrs := "server=" + p.Server
	return fmt.Sprintf("IMAP(%s)", attrs)
}

func (p *IMAP) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
	// validate
	if p.Server == "" {
		return p.err("server parameter is required")
	}

	if p.TLS {
		p.Server = util.EnsureAddressWithPort(p.Server, 993)
	} else {
		p.Server = util.EnsureAddressWithPort(p.Server, 143)
	}

	for {
		conn, err := util.SocketClient("tcp", p.Server, p.Timeout(), p.TLS, "")

		if err == nil {
			scanner := bufio.NewScanner(conn)
			if scanner.Scan() {
				t := scanner.Text()
				if len(t) < 4 {
					err = p.err("not enough data received from the server (received '%s')", t)
				}
				if !strings.HasPrefix(t, "* OK") {
					err = p.err("the server did not respond with an OK message")
				}
			}

			if err == nil {
				err = scanner.Err()
			}
		}

		if conn != nil {
			_, _ = conn.Write([]byte("001 LOGOUT\r\n"))
			_ = conn.Close()
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
