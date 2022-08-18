package probe

import (
	"context"
	"fmt"
	"net"
	"strings"
)

type DNSProbe struct {
	BaseProbe

	Server    string `yaml:"server"`
	QueryType string `yaml:"query_type"`
	Query     string `yaml:"query"`
}

func (p *DNSProbe) err(f string, args ...interface{}) error {
	return fmt.Errorf("dns: "+f, args...)
}

func (p *DNSProbe) String() string {
	attrs := "query=" + p.Query
	if p.QueryType != "" {
		attrs += ", query_type=" + p.QueryType
	}
	if p.Server != "" {
		attrs += ", server=" + p.Server
	}
	return fmt.Sprintf("Socket(%s)", attrs)
}

func (p *DNSProbe) RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error {
	// validate
	if p.Query == "" {
		return p.err("query parameter is required")
	}
	if p.QueryType == "" {
		p.QueryType = "A"
	}
	if p.Server != "" {
		p.Server = ensureAddressWithPort(p.Server, 53)
	}

	for {
		var err error

		r := &net.Resolver{}

		if p.Server != "" {
			r.PreferGo = true
			r.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: p.UpdateEvery(),
				}
				return d.DialContext(ctx, network, p.Server)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), p.UpdateEvery())

		switch strings.ToUpper(p.QueryType) {
		case "A", "AAAA":
			_, err = r.LookupHost(ctx, p.Query)
		case "CNAME":
			_, err = r.LookupCNAME(ctx, p.Query)
		case "ADDR":
			_, err = r.LookupAddr(ctx, p.Query)
		case "MX":
			_, err = r.LookupMX(ctx, p.Query)
		case "NS":
			_, err = r.LookupNS(ctx, p.Query)
		case "TXT":
			_, err = r.LookupTXT(ctx, p.Query)
		case "SRV":
			sq := strings.SplitN(p.Query, ".", 3)
			if len(sq) != 3 {
				err = p.err("SRV query must be in the form service.proto.name")
			} else {
				_, _, err = r.LookupSRV(ctx, sq[0], sq[1], sq[2])
			}
		default:
			err = p.err("unknown query type %s", p.QueryType)
		}

		cancel()

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
