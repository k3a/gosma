package probe

import (
	"bufio"
	"fmt"

	"github.com/k3a/gosma/util"
)

type SMTP struct {
	base

	Server string `yaml:"server"`
	TLS    bool   `yaml:"tls"`
}

func (p *SMTP) err(f string, args ...interface{}) error {
	return fmt.Errorf("smtp: "+f, args...)
}

func (p *SMTP) String() string {
	attrs := "server=" + p.Server
	return fmt.Sprintf("SMTP(%s)", attrs)
}

func (p *SMTP) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
	// validate
	if p.Server == "" {
		return p.err("server parameter is required")
	}
	p.Server = util.EnsureAddressWithPort(p.Server, 25)

	for {
		conn, err := util.SocketClient("tcp", p.Server, p.UpdateEvery(), p.TLS, "")

		if err == nil {
			scanner := bufio.NewScanner(conn)

			expect := func(code string) (ok bool, err error) {
				for scanner.Scan() {
					t := scanner.Text()
					if len(t) < 3 {
						return false, p.err("not enough data received from the server (received '%s')", t)
					}
					// skip multi-line response
					if len(t) > 3 && t[3] == '-' {
						continue
					}

					if t[:3] != code {
						return false, nil
					}

					return true, nil
				}
				return false, scanner.Err()
			}

			say := func(msg string) error {
				_, err = conn.Write([]byte(msg))
				return err
			}

			ok, err := expect("220")
			if err == nil {
				if !ok {
					err = p.err("did not receive an expected 220 welcome code")
				} else {
					err = say("EHLO localhost\r\n")
					if err == nil {
						ok, err = expect("250")
						if err == nil && !ok {
							err = say("HELO localhost\r\n")
							if err == nil {
								ok, err = expect("250")
							}
						}
						if err == nil && !ok {
							err = p.err("did not receive an expected 250 code after HELO")
						}
					}
				}
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
