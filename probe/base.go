package probe

import (
	"fmt"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type BaseProbe struct {
	name     string
	parent   Probe
	children ProbeMap

	YType        string        `yaml:"type"`
	YDescription string        `yaml:"description"`
	YUpdateEvery time.Duration `yaml:"update_every"`
}

func (bp *BaseProbe) nextRun(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return false
	case <-time.After(bp.UpdateEvery()):
		return true
	}
}

func (bp *BaseProbe) RunLoop(ch <-chan struct{}) error {
	return nil
}

func (bp *BaseProbe) Name() string {
	t := bp.name

	p := bp.Parent()
	for !reflect.ValueOf(p).IsNil() {
		t = p.Name() + "." + t
		p = p.Parent()
	}

	return t
}

func (bp *BaseProbe) Parent() Probe {
	return bp.parent
}

func (bp *BaseProbe) Type() string {
	return bp.YType
}

func (bp *BaseProbe) Description() string {
	return bp.YDescription
}

func (bp *BaseProbe) UpdateEvery() time.Duration {
	return bp.YUpdateEvery
}

func (bp *BaseProbe) Children() ProbeMap {
	return bp.children
}

func (bp *BaseProbe) String() string {
	if bp == nil {
		return "<nil>"
	}

	indent := ""
	p := bp.Parent()
	for !reflect.ValueOf(p).IsNil() {
		indent += " "
		p = p.Parent()
	}

	s := fmt.Sprintf("BaseProbe with %d children:\n", len(bp.children))

	for k, v := range bp.children {
		s += fmt.Sprintf("%s %s: %s\n", indent, k, v.String())
	}

	return s[:len(s)-1]
}

func instantiateBaseProbe(parent *BaseProbe, n *yaml.Node, name string) (*BaseProbe, error) {
	var bp BaseProbe

	// apply defautls by copying from the parent
	if parent != nil {
		bp = *parent
	}

	// overwrite stuff which is unique for every instance
	bp.parent = parent
	bp.name = name

	// apply additional defaults for zeros
	if bp.YUpdateEvery == 0 {
		bp.YUpdateEvery = 30 * time.Second
	}

	// potentially overwrite some by decoding from YAML
	err := n.Decode(&bp)
	if err != nil {
		return nil, err
	}

	return &bp, nil
}
