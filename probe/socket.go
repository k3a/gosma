package probe

import (
	"fmt"
	"io"
)

type SocketProbe struct {
	BaseProbe

	SocketType string `yaml:"socket_type"`
	Address    string `yaml:"address"`
	Path       string `yaml:"path"`
	TLS        bool   `yaml:"tls"`
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
			conn io.ReadCloser
			err  error
		)
		if p.SocketType == "unix" {
			conn, err = socketClient("unix", p.Path, p.UpdateEvery(), p.TLS)
		} else {
			conn, err = socketClient(p.SocketType, p.Address, p.UpdateEvery(), p.TLS)
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
