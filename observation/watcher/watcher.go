package watcher

import (
	"errors"

	"github.com/k3a/gosma/observation"
	"gopkg.in/yaml.v3"
)

type Watcher interface {
	observation.ProbeChangeObserver
	Init(n *yaml.Node) error
}

type base struct {
	Type string `yaml:"type"`
}

func WatcherFromYAMLNode(n *yaml.Node) (Watcher, error) {
	var bw base

	err := n.Decode(&bw)
	if err != nil {
		return nil, err
	}

	var w Watcher

	switch bw.Type {
	case "dashboard":
		w = &Dashboard{}
	default:
		return nil, errors.New("unknown watcher type " + bw.Type)
	}

	return w, w.Init(n)
}
