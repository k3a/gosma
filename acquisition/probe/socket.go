package probe

import (
	"fmt"
	"io"

	"github.com/k3a/gosma/util"
)

type Socket struct {
	base

	SocketType    string `yaml:"socket_type"`
	Address       string `yaml:"address"`
	Path          string `yaml:"path"`
	TLS           bool   `yaml:"tls"`
	TLSServerName string `yaml:"tls_server_name"`
}

func (p *Socket) err(f string, args ...interface{}) error {
	return fmt.Errorf("socket: "+f, args...)
}

func (p *Socket) String() string {
	attrs := "socket_type=" + p.SocketType
	if p.Address != "" {
		attrs += ", address=" + p.Address
	}
	if p.Path != "" {
		attrs += ", path=" + p.Path
	}
	if p.TLSServerName != "" {
		attrs += ", tls_server_name=" + p.TLSServerName
	}
	return fmt.Sprintf("Socket(%s)", attrs)
}

func (p *Socket) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
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
			conn, err = util.SocketClient("unix", p.Path, p.UpdateEvery(), p.TLS, p.TLSServerName)
		} else {
			conn, err = util.SocketClient(p.SocketType, p.Address, p.UpdateEvery(), p.TLS, p.TLSServerName)
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
