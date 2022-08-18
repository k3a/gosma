package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/k3a/gosma/alerter"
	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/probe"
	"github.com/k3a/gosma/watcher"
)

type Monitor struct {
	chStopProbes chan struct{}
	Probes       probe.ProbeMap `yaml:"-"`
	Watchers     []watcher.Watcher
	Alerters     []alerter.Alerter
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

	err := config.C.LoadFromFile(cpath)
	if err != nil {
		return err
	}

	m.Watchers = nil
	for _, n := range config.C.Watchers {
		w, err := watcher.WatcherFromYAMLNode(&n)
		errChk(err, "creating a watcher")

		m.Watchers = append(m.Watchers, w)
	}

	m.Alerters = nil
	for _, n := range config.C.Alerters {
		a, err := alerter.AlerterFromYAMLNode(&n)
		errChk(err, "creating an alerter")

		m.Alerters = append(m.Alerters, a)
	}

	m.Probes, err = probe.ParseProbesFromYAMLNode(nil, &config.C.Probes)
	if err != nil {
		return err
	}

	// print all probes
	fmt.Println(m.Probes)
	//os.Stdout.Sync()

	m.startProbes()

	return nil
}

func (m *Monitor) ProbeResult(p probe.Probe, d time.Duration, err error) {
	for _, w := range m.Watchers {
		go func(w watcher.Watcher) {
			err = w.HandleProbeResult(p, d, err)
			if err != nil {
				log.Printf("watcher error: %v", err)
			}
		}(w)
	}

	go func(alerters []alerter.Alerter) {
		alertedOk := false
		for _, w := range alerters {
			if !alertedOk || w.ShouldAlertAlways() {
				err = w.HandleProbeResult(p, d, err)
				if err != nil {
					log.Printf("alerter error: %v", err)
				} else {
					alertedOk = true
				}
			}
		}
	}(m.Alerters)
}

func errChk(err error, desc string) {
	if err != nil {
		fmt.Fprint(os.Stderr, "ERROR "+desc+": "+err.Error()+"\n")
		os.Exit(9)
	}
}

func runProbeLoopInBackground(p probe.Probe, stop <-chan struct{}, res probe.ProbeResultReceiver) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, false)
				log.Printf("PANIC: %v\n%s\n", r, buf)
			}
		}()

		err := p.RunLoop(stop, res)
		if err != nil {
			go res.ProbeResult(p, 0, err)
		}
	}()

	for _, cp := range p.Children() {
		runProbeLoopInBackground(cp, stop, res)
	}
}

func main() {
	fmt.Println(config.C.Name + " " + config.Buildstamp + "\n")

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
