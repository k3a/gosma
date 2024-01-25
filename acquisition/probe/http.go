package probe

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/k3a/gosma/config"
)

type HTTP struct {
	base
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Body    string            `yaml:"body"`
	Headers map[string]string `yaml:"headers"`

	ResponseMatch string `yaml:"response_match"`
	StatusCode    int    `yaml:"status_code"`

	inversed bool
}

func (p *HTTP) err(f string, args ...interface{}) error {
	return fmt.Errorf("http: "+f, args...)
}

func (p *HTTP) String() string {
	attrs := "url=" + p.URL
	if p.ResponseMatch != "" {
		attrs += ", response_match=" + p.ResponseMatch
	}
	if p.StatusCode != 0 {
		attrs += fmt.Sprintf(", status_code=%v", p.StatusCode)
	}
	return fmt.Sprintf("HTTP(%s)", attrs)
}

func (p *HTTP) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
	// validate
	if p.URL == "" {
		return p.err("parameter url is required")
	}
	if p.Method == "" {
		p.Method = "GET"
	}
	if p.StatusCode == 0 {
		p.StatusCode = 200
	}

	// request
	var body io.Reader
	if p.Body != "" {
		body = strings.NewReader(p.Body)
	}

	req, err := http.NewRequest(p.Method, p.URL, body)
	if err != nil {
		return p.err("error creating request: %v", err)
	}

	// headers
	req.Header.Set("User-Agent", config.C.Name)
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	// client
	cli := &http.Client{
		Timeout: p.Timeout(),
	}

	// inverse match?
	match_pattern := p.ResponseMatch
	p.inversed = strings.HasPrefix(match_pattern, "!")
	if p.inversed {
		match_pattern = match_pattern[1:]
	}

	for {
		resp, err := cli.Do(req)
		if err == nil {
			if resp.StatusCode != p.StatusCode {
				err = p.err("bad status code %d (expected %d)", resp.StatusCode, p.StatusCode)
			} else if match_pattern != "" {
				var bts []byte
				bts, err = io.ReadAll(resp.Body)
				if err == nil {
					var matched bool
					matched, err = regexp.MatchString(match_pattern, string(bts))
					if err == nil && matched == p.inversed {
						not := "not "
						if p.inversed {
							not = ""
						}
						err = p.err("pattern %s did %smatch the response body", match_pattern, not)
					}
				}
			}
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
