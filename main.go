package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/k3a/gosma/probe"

	"gopkg.in/yaml.v3"
)

type Config struct {
	YProbes yaml.Node      `yaml:"probes"`
	Probes  probe.ProbeMap `yaml:"-"`
}

func errChk(err error, desc string) {
	if err != nil {
		fmt.Fprint(os.Stderr, "ERROR "+desc+": "+err.Error()+"\n")
		os.Exit(9)
	}
}

func runProbeLoopInBackground(p probe.Probe, c *Config, ch <-chan struct{}) {
	go func() {
		err := p.RunLoop(ch)
		if err != nil {
			log.Printf("error running %s: %v\n", p.Name(), err)
		}
	}()

	for _, cp := range p.Children() {
		runProbeLoopInBackground(cp, c, ch)
	}
}

func main() {
	f, err := os.Open("testdata/ndemic.yaml")
	errChk(err, "opening file")

	y := yaml.NewDecoder(f)

	c := Config{}

	err = y.Decode(&c)
	errChk(err, "decoding config file")

	c.Probes, err = probe.ParseProbesFromYAMLNode(nil, &c.YProbes)
	errChk(err, "parsing probe nodes")

	// print all probes
	fmt.Println(c.Probes)

	// start all probes
	ch := make(chan struct{})
	for _, p := range c.Probes {
		runProbeLoopInBackground(p, &c, ch)
	}

	// to stop all
	// time.Sleep(3 * time.Second)
	// close(ch)

	err = http.ListenAndServe("0.0.0.0:1840", nil)
	errChk(err, "listening for http traffic")
}
