package alerter

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/probe"
	"gopkg.in/yaml.v3"
)

type ntfyServiceStatus struct {
	lastStateTransition time.Time
	currentState        string
	inited              bool
}

type NTFYAlerter struct {
	baseAlerter

	Server   string `yaml:"server"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Priority string `yaml:"priority"`
	Channel  string `yaml:"channel"`

	m      sync.RWMutex
	status map[string]*ntfyServiceStatus
}

func (w *NTFYAlerter) err(estr string, args ...interface{}) error {
	return fmt.Errorf("ntfy watcher: "+estr, args...)
}

func (w *NTFYAlerter) log(str string, args ...interface{}) {
	log.Printf("ntfy watcher: "+str, args...)
}

func (w *NTFYAlerter) Init(n *yaml.Node) error {
	w.status = make(map[string]*ntfyServiceStatus)

	err := n.Decode(&w)
	if err != nil {
		return err
	}

	if w.Server == "" {
		return w.err("server parameter URL is required")
	}
	if !strings.HasPrefix(w.Server, "http") {
		w.Server = "https://" + w.Server
	}

	if w.Channel == "" {
		return w.err("channel parameter is required")
	}
	if w.Priority == "" {
		w.Priority = "default"
	}

	w.log("will send alerts to %s", w.Server)

	return nil
}

func (w *NTFYAlerter) sendAlert(title, text string) error {
	cli := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", w.Server+"/"+w.Channel, strings.NewReader(text))
	if err != nil {
		return err
	}

	if w.Username != "" || w.Password != "" {
		req.SetBasicAuth(w.Username, w.Password)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("t", title)
	req.Header.Set("p", w.Priority)

	var resp *http.Response
	retry := 5
	sleep := 1
	for retry > 0 {
		resp, err = cli.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		} else if err != nil {
			w.log("error %v while sending notification for channel %s with title %s, priority %s",
				err, w.Channel, title, w.Priority)
		} else {
			err = w.err("bad status %d while sending notification", resp.StatusCode)
			w.log("bad status %d while sending notification for channel %s with title %s, priority %s",
				resp.StatusCode, w.Channel, title, w.Priority)
		}

		retry--

		if retry > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
			sleep *= 2
		}
	}

	return err
}

func (w *NTFYAlerter) HandleProbeResult(p probe.Probe, d time.Duration, perr error) (err error) {
	w.m.Lock()
	defer w.m.Unlock()

	name := p.Name()

	st, has := w.status[name]
	if !has {
		st = &ntfyServiceStatus{}
		w.status[name] = st
	}

	newState := "OK"
	newTitle := fmt.Sprintf("%s alert: %s OK", config.C.Name, name)
	newDesc := fmt.Sprintf("Service %s recovered after %s", name, time.Since(st.lastStateTransition).Round(time.Second))
	if perr != nil {
		newState = perr.Error()
		newTitle = fmt.Sprintf("%s alert: %s FAIL", config.C.Name, name)
		newDesc = fmt.Sprintf("Service %s failed with %s", name, newState)
	}

	if config.C.URL != "" {
		newDesc += "\n" + config.C.URL
	}

	// state change
	if st.currentState != newState {
		st.currentState = newState
		st.lastStateTransition = time.Now()

		// send notification
		if st.inited || newState != "OK" {
			err = w.sendAlert(newTitle, newDesc)
		}
	}

	st.inited = true

	return err
}
