package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/k3a/gosma/probe"
	"github.com/k3a/gosma/watcher"

	"gopkg.in/yaml.v3"
)

var buildstamp = "DEBUG"

type Config struct {
	YProbes   yaml.Node   `yaml:"probes"`
	YWatchers []yaml.Node `yaml:"watchers"`
}

type Monitor struct {
	chStopProbes chan struct{}
	Config       Config
	Probes       probe.ProbeMap `yaml:"-"`
	Watchers     []probe.ProbeResultReceiver
}

func (m *Monitor) stopProbes() {
	if m.chStopProbes != nil {
		close(m.chStopProbes)
		m.chStopProbes = nil
	}
}

func (m *Monitor) startProbes() {
	m.chStopProbes = make(chan struct{})

	for _, p := range m.Probes {
		runProbeLoopInBackground(p, m.chStopProbes, m)
	}
}

func (m *Monitor) LoadConfig(cpath string) error {
	m.stopProbes()

	f, err := os.Open(cpath)
	if err != nil {
		return err
	}

	y := yaml.NewDecoder(f)

	err = y.Decode(&m.Config)
	if err != nil {
		return err
	}

	m.Watchers = nil
	for _, n := range m.Config.YWatchers {
		w, err := watcher.WatcherFromYAMLNode(&n)
		errChk(err, "creating a watcher")

		m.Watchers = append(m.Watchers, w)
	}

	m.Probes, err = probe.ParseProbesFromYAMLNode(nil, &m.Config.YProbes)
	if err != nil {
		return err
	}

	// print all probes
	fmt.Println(m.Probes)

	m.startProbes()

	return nil
}

func (m *Monitor) ProbeResult(p probe.Probe, d time.Duration, err error) {
	for _, w := range m.Watchers {
		go w.ProbeResult(p, d, err)
	}
}

func errChk(err error, desc string) {
	if err != nil {
		fmt.Fprint(os.Stderr, "ERROR "+desc+": "+err.Error()+"\n")
		os.Exit(9)
	}
}

func runProbeLoopInBackground(p probe.Probe, stop <-chan struct{}, res probe.ProbeResultReceiver) {
	go func() {
		err := p.RunLoop(stop, res)
		if err != nil {
			log.Printf("error running %s: %v\n", p.Name(), err)
		}
	}()

	for _, cp := range p.Children() {
		runProbeLoopInBackground(cp, stop, res)
	}
}

func main() {
	fmt.Println("GoSMA " + buildstamp + "\n")

	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path-to-config>\n", os.Args[0])
		os.Exit(1)
	}
	configPath := os.Args[1]

	m := &Monitor{}

	err := m.LoadConfig(configPath)
	errChk(err, "loading config")

	select {} // sleep forever
}
