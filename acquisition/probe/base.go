package probe

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/k3a/gosma/config"
	"gopkg.in/yaml.v3"
)

type base struct {
	name     string
	parent   Probe
	children Map
	runStart time.Time

	YType          string        `yaml:"type"`            // type of the probe
	YDescription   string        `yaml:"description"`     // optional description of the probe or group
	YUpdateEvery   time.Duration `yaml:"update_every"`    // update (run a check) in this frequency
	YTimeout       time.Duration `yaml:"timeout"`         // probe timeout (same as update_every by default)
	YAlertAfter    int           `yaml:"alert_after"`     // alert after this many cycles
	YExecAfter     int           `yaml:"exec_after"`      // exec a command after this many cycles
	YExecOnSuccess string        `yaml:"exec_on_success"` // exec a command on transistion to the success state
	YExecOnFailure string        `yaml:"exec_on_failure"` // exec a command on transistion to the failed state
}

func (bp *base) nextRun(stop <-chan struct{}) bool {
	randExtra := float64(bp.UpdateEvery()) * (config.C.RandomizePerc / 100.0) * (rand.Float64() - 0.5)

	select {
	case <-stop:
		return false
	case <-time.After(bp.UpdateEvery() + time.Duration(randExtra)):
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
	if !reflect.ValueOf(p).IsNil() {
		t = p.Name() + "." + t
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

func (bp *base) Timeout() time.Duration {
	return bp.YTimeout
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

	// apply additional defaults
	if bp.YUpdateEvery == 0 {
		bp.YUpdateEvery = 30 * time.Second
	}
	if bp.YTimeout == 0 {
		bp.YTimeout = bp.YUpdateEvery
	}

	// potentially overwrite some fields by decoding from YAML
	err := n.Decode(&bp)
	if err != nil {
		return nil, err
	}

	return &bp, nil
}
