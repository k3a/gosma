package monitor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/k3a/gosma/acquisition/probe"
	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/observation"
	"github.com/k3a/gosma/observation/alerter"
	"github.com/k3a/gosma/observation/watcher"
)

// Monitor glues probes, watchers and alerters together
type Monitor struct {
	chStopProbes chan struct{}
	status       map[string]*observation.ProbeStatus
	startTime    time.Time
	m            sync.Mutex

	Probes   probe.Map `yaml:"-"`
	Watchers []watcher.Watcher
	Alerters []alerter.Alerter
}

func (m *Monitor) stopProbes() {
	if m.chStopProbes != nil {
		close(m.chStopProbes)
		m.chStopProbes = nil
	}

	// reset state
	m.startTime = time.Now()
}

func (m *Monitor) startProbes() {
	m.chStopProbes = make(chan struct{})

	for _, p := range m.Probes {
		runProbeLoopInBackground(p, m.chStopProbes, m)
	}
}

func (m *Monitor) LoadConfig(cpath string) error {
	m.stopProbes()

	m.status = make(map[string]*observation.ProbeStatus)

	err := config.C.LoadFromFile(cpath)
	if err != nil {
		return err
	}

	m.Watchers = nil
	for _, n := range config.C.Watchers {
		w, err := watcher.WatcherFromYAMLNode(&n)
		if err != nil {
			return fmt.Errorf("error creating a watcher: %w", err)
		}

		m.Watchers = append(m.Watchers, w)
	}

	m.Alerters = nil
	for _, n := range config.C.Alerters {
		a, err := alerter.FromYAMLNode(&n)
		if err != nil {
			return fmt.Errorf("error creating an alerter: %w", err)
		}

		m.Alerters = append(m.Alerters, a)
	}

	m.Probes, err = probe.FromYAMLNode(nil, &config.C.Probes)
	if err != nil {
		return err
	}

	// print all probes
	fmt.Println(m.Probes)
	//os.Stdout.Sync()

	if config.C.Delay != 0 {
		log.Printf("Notifications and execs will be muted for %s", config.C.Delay)
	}

	m.startProbes()

	return nil
}

func (m *Monitor) execOnStateChange(p probe.Probe, state string, numCycles int) {
	var cmd *exec.Cmd

	thisCycle := ((p.ExecAfter() == 0 && numCycles == 1) || (p.ExecAfter() > 0 && p.ExecAfter() == numCycles))
	if !thisCycle {
		return
	}

	name := p.Name()

	if state == "OK" && p.ExecOnSuccess() != "" {
		log.Printf("exec_on_success for %s: %s", name, p.ExecOnSuccess())
		cmd = exec.Command("sh", "-c", p.ExecOnSuccess())
	} else if state != "OK" && p.ExecOnFailure() != "" {
		log.Printf("exec_on_failure for %s: %s", name, p.ExecOnFailure())
		cmd = exec.Command("sh", "-c", p.ExecOnFailure())
	}

	if cmd != nil {
		go func(cmd *exec.Cmd) {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Printf("error executing %s %v: %v", cmd.Path, cmd.Args, err)
			}
		}(cmd)
	}

}

// ProbeResult is called from a probe
func (m *Monitor) ProbeResult(p probe.Probe, d time.Duration, err error) {
	m.m.Lock()
	defer m.m.Unlock()

	name := p.Name()

	st, has := m.status[name]
	if !has {
		st = &observation.ProbeStatus{}
		m.status[name] = st
	}

	st.Probe = p
	st.Latency = d

	newState := "OK"
	if err != nil {
		newState = err.Error()
	}

	const initialState = "OK" // the initial state we expect is OK

	// state change
	if st.CurrentState != newState {
		if st.CurrentState == "" {
			st.CurrentState = initialState
			st.LastStateTransition = time.Now()
		}
		st.PreviousState = st.CurrentState
		st.CurrentState = newState
		st.NumCyclesInCurrentState = 0
	}

	// delay from startup
	if config.C.Delay != 0 && !st.Unmuted {
		if time.Since(m.startTime) < config.C.Delay {
			// currently muted, set previous state to the current state to avoid alerts
			st.PreviousState = st.CurrentState
		} else {
			// just unmuted, set the previous state to the initial
			st.Unmuted = true
			st.PreviousState = initialState
			st.NumCyclesInCurrentState = 0
		}
	}

	st.NumCyclesInCurrentState += 1

	// watchers (for every change)
	for _, w := range m.Watchers {
		go func(w watcher.Watcher, st observation.ProbeStatus) {
			err = w.ProbeStateChanged(p, st)
			if err != nil {
				log.Printf("watcher error: %v", err)
			}
		}(w, *st)
	}

	// alerters (only for state transitions)
	go func(alerters []alerter.Alerter, st observation.ProbeStatus) {
		alertedOk := false
		for _, w := range alerters {
			if !alertedOk || w.ShouldAlertAlways() {
				err = w.ProbeStateChanged(p, st)
				if err != nil {
					log.Printf("alerter error: %v", err)
				} else {
					alertedOk = true
				}
			}
		}
	}(m.Alerters, *st)

	// stuff to do for a state change after watchers/alerters are processed
	// code in this block is also not run during "muted" period caused by config.C.delay
	if st.PreviousState != st.CurrentState {
		// run execs on state change
		m.execOnStateChange(p, st.CurrentState, st.NumCyclesInCurrentState)

		// Record timetamp of this state change as the last thing.
		// This has to be done _after_ running watchers/alerters/execs as they need to know
		// the timestamp of the previous state change not the current one which is "now"
		if st.NumCyclesInCurrentState == 1 {
			st.LastStateTransition = time.Now()
		}
	}
}

func runProbeLoopInBackground(p probe.Probe, stop <-chan struct{}, res probe.ResultReceiver) {
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
