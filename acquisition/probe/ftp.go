package probe

import (
	"bufio"
	"fmt"

	"github.com/k3a/gosma/util"
)

type FTP struct {
	base

	Server string `yaml:"server"`
	TLS    bool   `yaml:"tls"`
}

func (p *FTP) err(f string, args ...any) error {
	return fmt.Errorf("ftp: "+f, args...)
}

func (p *FTP) String() string {
	attrs := "server=" + p.Server
	return fmt.Sprintf("FTP(%s)", attrs)
}

func (p *FTP) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
	// validate
	if p.Server == "" {
		return p.err("server parameter is required")
	}
	p.Server = util.EnsureAddressWithPort(p.Server, 21)

	for {
		conn, err := util.SocketClient("tcp", p.Server, p.Timeout(), p.TLS, "")

		if err == nil {
			scanner := bufio.NewScanner(conn)
			ok := false
			for scanner.Scan() {
				t := scanner.Text()
				if len(t) < 3 {
					break
				}
				// skip multi-line response
				if len(t) > 3 && t[3] == '-' {
					continue
				}

				if t[:3] != "220" {
					err = p.err("received unexpected data '%s' instead of a status code", t[:3])
				}

				ok = true
				break
			}

			if !ok {
				err = p.err("not enough data received (received '%s')", scanner.Text())
			}

			if err == nil {
				err = scanner.Err()
			}
		}

		if conn != nil {
			_, _ = conn.Write([]byte("QUIT\r\n"))
			_ = conn.Close()
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
