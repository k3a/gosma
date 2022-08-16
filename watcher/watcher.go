package watcher

import (
	"errors"

	"github.com/k3a/gosma/probe"
	"gopkg.in/yaml.v3"
)

type baseWatcher struct {
	Type string `yaml:"type"`
}

type Watcher interface {
	probe.ProbeResultReceiver
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
	case "ntfy":
		w = &NTFYWatcher{}
	default:
		return nil, errors.New("unknown watcher type " + bw.Type)
	}

	return w, w.Init(n)
}
