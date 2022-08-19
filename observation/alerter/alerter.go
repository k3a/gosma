package alerter

import (
	"errors"

	"github.com/k3a/gosma/acquisition/probe"
	"github.com/k3a/gosma/observation"
	"gopkg.in/yaml.v3"
)

type base struct {
	Type        string `yaml:"type"`
	AlertAlways bool   `yaml:"alert_always"`
}

func (ba *base) ShouldAlertAlways() bool {
	return ba.AlertAlways
}

func (ba *base) shouldAlert(p probe.Probe, st observation.ProbeStatus) bool {
	if st.PreviousState == st.CurrentState {
		// ignore initial state after startup
		return false
	}
	aa := p.AlertAfter()
	return (aa == 0 && st.NumCyclesInCurrentState == 1) || (aa > 0 && st.NumCyclesInCurrentState == aa)
}

type Alerter interface {
	observation.ProbeChangeObserver
	ShouldAlertAlways() bool
	Init(n *yaml.Node) error
}

func FromYAMLNode(n *yaml.Node) (Alerter, error) {
	var ba base

	err := n.Decode(&ba)
	if err != nil {
		return nil, err
	}

	var a Alerter

	switch ba.Type {
	case "ntfy":
		a = &NTFY{}
	case "smtp":
		a = &SMTP{}
	default:
		return nil, errors.New("unknown alerter type " + ba.Type)
	}

	return a, a.Init(n)
}
