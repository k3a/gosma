package watcher

import (
	"errors"
	"time"

	"github.com/k3a/gosma/probe"
	"gopkg.in/yaml.v3"
)

type baseWatcher struct {
	Type string `yaml:"type"`
}

type Watcher interface {
	// HandleProbeResult is called by a probe in a new gorountine
	HandleProbeResult(p probe.Probe, d time.Duration, perr error) error
	Init(n *yaml.Node) error
}

func WatcherFromYAMLNode(n *yaml.Node) (Watcher, error) {
	var bw baseWatcher

	err := n.Decode(&bw)
	if err != nil {
		return nil, err
	}

	var w Watcher

	switch bw.Type {
	case "dashboard":
		w = &DashboardWatcher{}
	default:
		return nil, errors.New("unknown watcher type " + bw.Type)
	}

	return w, w.Init(n)
}
