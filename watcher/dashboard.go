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
	latency             time.Duration
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
			background-color: #000;
			color: #fff;
		}
		.box {
			margin: 10px;
			background-color: #222;
			display: inline-block;
			min-width: 150px;
			padding: 5px;
			white-space: nowrap;
		}
		.box-wide {
		   min-width: 400px;
		}
	</style>`
	str += "</head><body>\n"

	if len(w.status) == 0 {
		str += "No probes defined"
	} else {
		//str += "<ul>\n"

		var keys []string
		for k := range w.status {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, name := range keys {
			st := w.status[name]

			col := "#00ff00"
			extraCls := ""
			extraText := fmt.Sprintf(", latency %s", st.latency.Round(time.Millisecond))
			if st.currentState != "OK" {
				col = "#ff0000"
				extraCls = "box-wide"
				extraText = ""
			}

			since := time.Since(st.lastStateTransition).Round(time.Second)

			str += fmt.Sprintf("<div class=\"box %s\" title=\"%s\"><b>%s</b><br><span style=\"color:%s\">%s</span> since %s%s</div>\n",
				extraCls, st.desc, name, col, st.currentState, since, extraText)
		}

		//str += "</ul>\n"
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

	st.latency = d

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
