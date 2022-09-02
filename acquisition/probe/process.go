package probe

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

const procBase = "/proc"

type Process struct {
	base

	Process string `yaml:"process"`
}

func (p *Process) err(f string, args ...interface{}) error {
	return fmt.Errorf("process: "+f, args...)
}

func (p *Process) String() string {
	attrs := "process=" + p.Process
	return fmt.Sprintf("Process(%s)", attrs)
}

func (p *Process) RunLoop(stop <-chan struct{}, res ResultReceiver) error {
	// validate
	if p.Process == "" {
		return p.err("process parameter is required for process socket type")
	}

	for {
		d, err := os.Open(procBase)

		found := false

	searchLoop:
		for {
			dirs, err := d.Readdirnames(512)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break searchLoop
			}

			for _, name := range dirs {
				if name[0] < '0' || name[0] > '9' {
					// only numeric
					continue
				}

				comm, err := os.ReadFile(path.Join(procBase, name, "comm"))
				commStr := strings.TrimSpace(string(comm))
				if err == nil && commStr == p.Process {
					found = true
					break searchLoop
				}
			}
		}

		if d != nil {
			_ = d.Close()
		}

		if !found {
			err = fmt.Errorf("process %s is not running", p.Process)
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
