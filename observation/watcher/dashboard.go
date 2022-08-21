package watcher

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/k3a/gosma/acquisition/probe"
	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/observation"
	"github.com/k3a/gosma/util"
	"gopkg.in/yaml.v3"
)

type dashboardServiceStatus struct {
	lastStateTransition time.Time
	currentState        string
	desc                string
	latency             time.Duration
}

type Dashboard struct {
	base

	YListen string `yaml:"listen"`

	srv                     *http.Server
	m                       sync.RWMutex
	status                  map[string]*dashboardServiceStatus
	lowestUpdateIntervalSec int
}

func (w *Dashboard) err(estr string, args ...interface{}) error {
	return fmt.Errorf("dashboard watcher: "+estr, args...)
}

func (w *Dashboard) log(str string, args ...interface{}) {
	log.Printf("dashboard watcher: "+str, args...)
}

func (w *Dashboard) Init(n *yaml.Node) error {
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

func (w *Dashboard) indexHandler(wr http.ResponseWriter, r *http.Request) {
	str := ""

	if r.URL.Query().Get("content") != "1" {
		str = `<!DOCTYPE html><html><head><title>` + config.C.Name + `</title>
		<meta name="viewport" content="width=800, initial-scale=1">
		<script>
		function ensureIframeLoaded() {
			var iframe = document.getElementById('content');

			// var refresh = false;
			// try {
			// 	refresh = (iframe.contentDocument || iframe.contentWindow.document) == null;
			// } catch(e) {
			// 	refresh = true;
			// }
			// if (refresh) {

			// reload iframe
			iframe.src += "";

			window.setTimeout(ensureIframeLoaded, ` + strconv.Itoa(w.lowestUpdateIntervalSec) + `000);
		}
		</script>
		<style>
		body, html {
			background-color:#000;margin:0;padding:0;height:100%;
		}
		iframe {
			width:100%;height:100%;border:0;
		}
		</style>
		<body onload="ensureIframeLoaded();">
		<iframe id="content" src=".?content=1">
		</iframe></body></html>`
	} else {
		w.m.RLock()
		defer w.m.RUnlock()

		str = "<!DOCTYPE html><html><head>\n"
		str += "<title>" + config.C.Name + "</title>\n"
		//str += fmt.Sprintf("<meta http-equiv=\"refresh\" content=\"%d\">\n", w.lowestUpdateIntervalSec)
		str += `<style>
		body {
			background-color: #000;
			color: #fff;
			font-family: Arial, Helvetica, sans-serif;
			font-size: 12pt;
			margin:0;
		}
		a {
			color: #ddf;
		}
		a:visited {
			color: #bbf;
		}
		a:hover {
			color: #ccf;
		}
		.title {
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
			max-width: 230px;
			display: inline-block;
			font-weight: bold;
		}
		.box-wide .title {
			max-width: 400px;
		}
		.box {
			margin: 5px;
			background-color: #222;
			display: inline-block;
			min-width: 230px;
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

				since := util.CompactDuration(time.Since(st.lastStateTransition), 2)

				str += fmt.Sprintf("<div class=\"box %s\" title=\"%s\"><div class=\"title\" title=\"%s\">%s</div><br><span style=\"color:%s\">%s</span> for %s%s</div>\n",
					extraCls, st.desc, name, name, col, st.currentState, since, extraText)
			}

			//str += "</ul>\n"
		}

		str += "<br><br><center><small>" + config.Footer + "</small></center>"
		str += "</body></html>"
	}

	wr.Header().Set("Content-Type", "text/html; charset=UTF-8")
	wr.Write([]byte(str))
}

func (w *Dashboard) ProbeStateChanged(p probe.Probe, ost observation.ProbeStatus) error {
	w.m.Lock()
	defer w.m.Unlock()

	name := p.Name()

	st, has := w.status[name]
	if !has {
		st = &dashboardServiceStatus{}
		w.status[name] = st
	}

	st.latency = ost.Latency
	st.currentState = ost.CurrentState
	st.lastStateTransition = ost.LastStateTransition

	if st.desc == "" {
		st.desc = p.Description()
	}

	updateIntervalSec := int(p.UpdateEvery() / time.Second)
	if w.lowestUpdateIntervalSec == 1 || (updateIntervalSec < w.lowestUpdateIntervalSec && updateIntervalSec > 1) {
		w.lowestUpdateIntervalSec = updateIntervalSec
	}

	return nil
}
