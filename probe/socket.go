package probe

import (
	"fmt"
	"net"
)

type SocketProbe struct {
	BaseProbe

	SocketType string `yaml:"socket_type"`
	Address    string `yaml:"address"`
	Path       string `yaml:"path"`
}

func (p *SocketProbe) err(f string, args ...interface{}) error {
	return fmt.Errorf("socket: "+f, args...)
}

func (p *SocketProbe) String() string {
	attrs := "socket_type=" + p.SocketType
	if p.Address != "" {
		attrs += ", address=" + p.Address
	}
	if p.Path != "" {
		attrs += ", path=" + p.Path
	}
	return fmt.Sprintf("Socket(%s)", attrs)
}

func (p *SocketProbe) RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error {
	// validate
	if p.SocketType == "" {
		p.SocketType = "tcp"
	}
	if p.SocketType == "unix" && p.Path == "" {
		return p.err("path parameter is required for unix socket type")
	}
	if p.SocketType != "unix" && p.Address == "" {
		return p.err("address parameter is required")
	}

	for {
		var (
			err  error
			conn net.Conn
		)
		if p.SocketType == "unix" {
			conn, err = net.Dial("unix", p.Path)
		} else {
			conn, err = net.Dial(p.SocketType, p.Address)
		}

		if err == nil {
			conn.Close()
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
