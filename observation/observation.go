package observation

import (
	"time"

	"github.com/k3a/gosma/acquisition/probe"
)

type ProbeStatus struct {
	Probe                   probe.Probe
	LastStateTransition     time.Time
	NumCyclesInCurrentState int
	CurrentState            string
	// PreviousState is the state before the most recent state transition
	PreviousState string
	Latency       time.Duration

	// Unmuted is used internally for delayed notifications and execs
	Unmuted bool
}

type ProbeChangeObserver interface {
	// ProbeStateChanged is called by the monitor in a new gorountine
	ProbeStateChanged(p probe.Probe, st ProbeStatus) error
}
