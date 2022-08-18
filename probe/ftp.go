package probe

import (
	"bufio"
	"fmt"

	"github.com/k3a/gosma/util"
)

type FTPProbe struct {
	BaseProbe

	Server string `yaml:"server"`
	TLS    bool   `yaml:"tls"`
}

func (p *FTPProbe) err(f string, args ...interface{}) error {
	return fmt.Errorf("ftp: "+f, args...)
}

func (p *FTPProbe) String() string {
	attrs := "server=" + p.Server
	return fmt.Sprintf("Socket(%s)", attrs)
}

func (p *FTPProbe) RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error {
	// validate
	if p.Server == "" {
		return p.err("server parameter is required")
	}
	p.Server = util.EnsureAddressWithPort(p.Server, 21)

	for {
		conn, err := util.SocketClient("tcp", p.Server, p.UpdateEvery(), p.TLS, "")

		if err == nil {
			scanner := bufio.NewScanner(conn)
			ok := false
			for scanner.Scan() {
				t := scanner.Text()
				if len(t) < 4 {
					break
				}
				// skip multi-line response
				if t[3] == '-' {
					continue
				}

				if t[:3] != "220" {
					err = p.err("received unexpected data '%s' instead of a status code", t[:3])
				}

				ok = true
				break
			}

			if !ok {
				err = p.err("not enough data received from the server")
			}

			if err == nil {
				err = scanner.Err()
			}
		}

		if conn != nil {
			conn.Write([]byte("QUIT\r\n"))
			conn.Close()
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
