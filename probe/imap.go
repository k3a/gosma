package probe

import (
	"bufio"
	"fmt"
	"strings"
)

type IMAPProbe struct {
	BaseProbe

	Server string `yaml:"server"`
	TLS    bool   `yaml:"tls"`
}

func (p *IMAPProbe) err(f string, args ...interface{}) error {
	return fmt.Errorf("imap: "+f, args...)
}

func (p *IMAPProbe) String() string {
	attrs := "server=" + p.Server
	return fmt.Sprintf("IMAP(%s)", attrs)
}

func (p *IMAPProbe) RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error {
	// validate
	if p.Server == "" {
		return p.err("server parameter is required")
	}

	if p.TLS {
		p.Server = ensureAddressWithPort(p.Server, 993)
	} else {
		p.Server = ensureAddressWithPort(p.Server, 143)
	}

	for {
		conn, err := socketClient("tcp", p.Server, p.UpdateEvery(), p.TLS, "")

		if err == nil {
			scanner := bufio.NewScanner(conn)
			if scanner.Scan() {
				t := scanner.Text()
				if len(t) < 4 {
					err = p.err("not enough data received from the server")
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
			conn.Write([]byte("001 LOGOUT\r\n"))
			conn.Close()
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
