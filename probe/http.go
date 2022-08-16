package probe

import (
	"fmt"
	"log"
)

type HTTPProbe struct {
	BaseProbe
	URL           string `yaml:"url"`
	ResponseMatch string `yaml:"response_match"`
}

func (p *HTTPProbe) RunLoop(stop <-chan struct{}) error {
	// validate
	if p.URL == "" {
		return fmt.Errorf("HTTPProbe: url is required")
	}

	for {
		if !p.nextRun(stop) {
			break
		}

		log.Printf("%s: works", p.Name())
	}

	return nil
}

func (p *HTTPProbe) String() string {
	attrs := "url=" + p.URL
	if p.ResponseMatch != "" {
		attrs += ", response_match=" + p.ResponseMatch
	}
	attrs += ", name=" + p.Name()
	return fmt.Sprintf("HTTPProbe(%s)", attrs)
}
