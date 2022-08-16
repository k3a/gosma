package watcher

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/k3a/gosma/probe"
	"gopkg.in/yaml.v3"
)

type dashboardServiceStatus struct {
	lastStateTransition time.Time
	currentState        string
	desc                string
}

type DashboardWatcher struct {
	YListen string `yaml:"listen"`

	srv                     *http.Server
	m                       sync.RWMutex
	status                  map[string]*dashboardServiceStatus
	lowestUpdateIntervalSec int
}

func (w *DashboardWatcher) err(estr string, args ...interface{}) error {
	return fmt.Errorf("dashboard watcher: "+estr, args...)
}

func (w *DashboardWatcher) log(str string, args ...interface{}) {
	log.Printf("dashboard watcher: "+str, args...)
}

func (w *DashboardWatcher) Init(n *yaml.Node) error {
	w.status = make(map[string]*dashboardServiceStatus)
	w.lowestUpdateIntervalSec = 1

	err := n.Decode(&w)
	if err != nil {
		return err
	}

	if w.YListen == "" {
		return w.err("listen parameter in the form IP:PORT is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", w.indexHandler)

	go func() {
		err := http.ListenAndServe(w.YListen, mux)
		if err != nil {
			w.log("error serving: %v", err)
		}
	}()

	w.log("listening on %s", w.YListen)

	return nil
}

func (w *DashboardWatcher) indexHandler(wr http.ResponseWriter, r *http.Request) {
	w.m.RLock()
	defer w.m.RUnlock()

	str := "<html><head>\n"
	str += "<title>GoSMA</title>\n"
	str += fmt.Sprintf("<meta http-equiv=\"refresh\" content=\"%d\">\n", w.lowestUpdateIntervalSec)
	str += `<style>
		body {
			background-color: #111;
			color: #fff;
		}
	</style>`
	str += "</head><body>\n"

	if len(w.status) == 0 {
		str += "No probes defined"
	} else {
		str += "<ul>\n"

		var keys []string
		for k := range w.status {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, name := range keys {
			st := w.status[name]

			col := "#00ff00"
			if st.currentState != "OK" {
				col = "#ff0000"
			}

			since := time.Since(st.lastStateTransition).Round(time.Second)

			str += fmt.Sprintf("<li><b>%s</b> <small>%s</small><br><span style=\"color:%s\">%s</span> since %s</li>\n",
				name, st.desc, col, st.currentState, since)
		}

		str += "</ul>\n"
	}

	str += "</head></body>"

	wr.Header().Set("Content-Type", "text/html; charset=UTF-8")
	wr.Write([]byte(str))
}

func (w *DashboardWatcher) ProbeResult(p probe.Probe, d time.Duration, err error) {
	w.m.Lock()
	defer w.m.Unlock()

	name := p.Name()

	st, has := w.status[name]
	if !has {
		st = &dashboardServiceStatus{}
		w.status[name] = st
	}

	newState := "OK"
	if err != nil {
		newState = err.Error()
	}

	// state change
	if st.currentState != newState {
		st.currentState = newState
		st.lastStateTransition = time.Now()
	}

	if st.desc == "" {
		st.desc = p.Description()
	}

	updateIntervalSec := int(p.UpdateEvery() / time.Second)
	if w.lowestUpdateIntervalSec == 1 || (updateIntervalSec < w.lowestUpdateIntervalSec && updateIntervalSec > 1) {
		w.lowestUpdateIntervalSec = updateIntervalSec
	}
}
