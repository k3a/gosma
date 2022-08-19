package probe

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type base struct {
	name     string
	parent   Probe
	children Map
	runStart time.Time

	YType          string        `yaml:"type"`
	YDescription   string        `yaml:"description"`
	YUpdateEvery   time.Duration `yaml:"update_every"`
	YAlertAfter    int           `yaml:"alert_after"`     // alert after this many cycles
	YExecAfter     int           `yaml:"exec_after"`      // exec a command after this many cycles
	YExecOnSuccess string        `yaml:"exec_on_success"` // exec a command on transistion to the success state
	YExecOnFailure string        `yaml:"exec_on_failure"` // exec a command on transistion to the failed state
}

func (bp *base) nextRun(stop <-chan struct{}) bool {
	randExtra := time.Duration(rand.Intn(200)-100) * time.Millisecond

	select {
	case <-stop:
		return false
	case <-time.After(bp.UpdateEvery() + randExtra):
		bp.runStart = time.Now()
		return true
	}
}

func (bp *base) runDuration() time.Duration {
	if bp.runStart.IsZero() {
		return 0
	}
	return time.Since(bp.runStart)
}

func (bp *base) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
	return nil
}

func (bp *base) Name() string {
	t := bp.name

	p := bp.Parent()
	for !reflect.ValueOf(p).IsNil() {
		t = p.Name() + "." + t
		p = p.Parent()
	}

	return t
}

func (bp *base) Parent() Probe {
	return bp.parent
}

func (bp *base) Type() string {
	return bp.YType
}

func (bp *base) Description() string {
	return bp.YDescription
}

func (bp *base) UpdateEvery() time.Duration {
	return bp.YUpdateEvery
}

func (bp *base) AlertAfter() int {
	return bp.YAlertAfter
}

func (bp *base) ExecAfter() int {
	return bp.YExecAfter
}

func (bp *base) ExecOnSuccess() string {
	return bp.YExecOnSuccess
}

func (bp *base) ExecOnFailure() string {
	return bp.YExecOnFailure
}

func (bp *base) Children() Map {
	return bp.children
}

func (bp *base) String() string {
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

func instantiateBaseProbe(parent *base, n *yaml.Node, name string) (*base, error) {
	var bp base

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
