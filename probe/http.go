package probe

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

const DefaultUserAgent = "gosma"

type HTTPProbe struct {
	BaseProbe
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Body    string            `yaml:"body"`
	Headers map[string]string `yaml:"headers"`

	ResponseMatch string `yaml:"response_match"`
	StatusCode    int    `yaml:"status_code"`
}

func (p *HTTPProbe) err(f string, args ...interface{}) error {
	return fmt.Errorf("http: "+f, args...)
}

func (p *HTTPProbe) String() string {
	attrs := "url=" + p.URL
	if p.ResponseMatch != "" {
		attrs += ", response_match=" + p.ResponseMatch
	}
	if p.StatusCode != 0 {
		attrs += fmt.Sprintf(", status_code=%v", p.StatusCode)
	}
	return fmt.Sprintf("HTTPProbe(%s)", attrs)
}

func (p *HTTPProbe) RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error {
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
	req.Header.Set("User-Agent", DefaultUserAgent)
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	// client
	cli := &http.Client{
		Timeout: p.UpdateEvery(),
	}

	for {
		resp, err := cli.Do(req)
		if err == nil {
			if resp.StatusCode != p.StatusCode {
				err = p.err("bad status code %d (expected %d)", resp.StatusCode, p.StatusCode)
			} else if p.ResponseMatch != "" {
				var bts []byte
				bts, err = ioutil.ReadAll(resp.Body)
				if err == nil {
					var matched bool
					matched, err = regexp.MatchString(p.ResponseMatch, string(bts))
					if err == nil && !matched {
						err = p.err("pattern %s did not match the response body", p.ResponseMatch)
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
