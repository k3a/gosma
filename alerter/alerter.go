package alerter

import (
	"errors"

	"github.com/k3a/gosma/watcher"
	"gopkg.in/yaml.v3"
)

type baseAlerter struct {
	Type        string `yaml:"type"`
	AlertAlways bool   `yaml:"alert_always"`
}

func (ba *baseAlerter) ShouldAlertAlways() bool {
	return ba.AlertAlways
}

type Alerter interface {
	watcher.Watcher
	ShouldAlertAlways() bool
}

func AlerterFromYAMLNode(n *yaml.Node) (Alerter, error) {
	var ba baseAlerter

	err := n.Decode(&ba)
	if err != nil {
		return nil, err
	}

	var a Alerter

	switch ba.Type {
	case "ntfy":
		a = &NTFYAlerter{}
	case "smtp":
		a = &SMTPAlerter{}
	default:
		return nil, errors.New("unknown alerter type " + ba.Type)
	}

	return a, a.Init(n)
}
