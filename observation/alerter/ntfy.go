package alerter

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/k3a/gosma/acquisition/probe"
	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/observation"
	"gopkg.in/yaml.v3"
)

type NTFY struct {
	base

	Server     string `yaml:"server"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Priority   string `yaml:"priority"`
	PriorityOK string `yaml:"priority_ok"`
	Channel    string `yaml:"channel"`
}

func (a *NTFY) err(estr string, args ...interface{}) error {
	return fmt.Errorf("ntfy watcher: "+estr, args...)
}

func (a *NTFY) log(str string, args ...interface{}) {
	log.Printf("ntfy watcher: "+str, args...)
}

func (a *NTFY) Init(n *yaml.Node) error {
	err := n.Decode(&a)
	if err != nil {
		return err
	}

	if a.Server == "" {
		return a.err("server parameter URL is required")
	}
	if !strings.HasPrefix(a.Server, "http") {
		a.Server = "https://" + a.Server
	}

	if a.Channel == "" {
		return a.err("channel parameter is required")
	}
	if a.Priority == "" {
		a.Priority = "default"
	}

	a.log("will send alerts to %s", a.Server)

	return nil
}

func (a *NTFY) sendAlert(title, text, priority string) error {
	cli := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", a.Server+"/"+a.Channel, nil)
	if err != nil {
		return err
	}

	if a.Username != "" || a.Password != "" {
		req.SetBasicAuth(a.Username, a.Password)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("t", title)
	req.Header.Set("p", priority)

	var resp *http.Response
	retry := 5
	sleep := 1
	for retry > 0 {
		req.Body = io.NopCloser(strings.NewReader(text))
		resp, err = cli.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		} else if err != nil {
			a.log("error %v while sending notification for channel %s with title %s, priority %s",
				err, a.Channel, title, a.Priority)
		} else {
			err = a.err("bad status %d while sending notification", resp.StatusCode)
			a.log("bad status %d while sending notification for channel %s with title %s, priority %s",
				resp.StatusCode, a.Channel, title, a.Priority)
		}

		retry--

		if retry > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
			sleep *= 2
		}
	}

	return err
}

func (a *NTFY) ProbeStateChanged(p probe.Probe, st observation.ProbeStatus) (err error) {
	name := p.Name()

	title := ""
	desc := ""
	priority := a.Priority

	if st.CurrentState == "OK" {
		title = fmt.Sprintf("%s alert: %s OK", config.C.Name, name)
		desc = fmt.Sprintf("%s recovered after %s", name, time.Since(st.LastStateTransition).Round(time.Second))
		if a.PriorityOK != "" {
			priority = a.PriorityOK
		}
	} else {
		title = fmt.Sprintf("%s alert: %s FAIL", config.C.Name, name)
		desc = fmt.Sprintf("%s failed with %s", name, st.CurrentState)
	}

	if config.C.URL != "" {
		desc += "\n" + config.C.URL
	}

	// send notification
	if a.shouldAlert(p, st) {
		go func() {
			err = a.sendAlert(title, desc, priority)
			if err != nil {
				log.Printf("ntfy alerter: error %v", err)
			}
		}()
	}

	return nil
}
