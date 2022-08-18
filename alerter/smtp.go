package alerter

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/probe"
	"github.com/k3a/gosma/util"
	"gopkg.in/yaml.v3"
)

type smtpServiceStatus struct {
	lastStateTransition time.Time
	currentState        string
	inited              bool
}

type SMTPAlerter struct {
	baseAlerter

	Server     string `yaml:"server"`
	From       string `yaml:"from"`
	To         string `yaml:"to"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Identity   string `yaml:"identity"`
	CRMMD5     bool   `yaml:"crammd5"`
	DisableTLS bool   `yaml:"disable_tls"`

	m      sync.RWMutex
	status map[string]*smtpServiceStatus
}

func (a *SMTPAlerter) err(estr string, args ...interface{}) error {
	return fmt.Errorf("smtp alerter: "+estr, args...)
}

func (a *SMTPAlerter) log(str string, args ...interface{}) {
	log.Printf("smtp alerter: "+str, args...)
}

func (a *SMTPAlerter) Init(n *yaml.Node) error {
	a.status = make(map[string]*smtpServiceStatus)

	err := n.Decode(&a)
	if err != nil {
		return err
	}

	if a.Server == "" {
		return a.err("server parameter is required")
	}
	a.Server = util.EnsureAddressWithPort(a.Server, 25)
	if a.To == "" {
		return a.err("to parameter is required")
	}
	if a.From == "" {
		a.From = config.C.Name + "@localhost"
	}

	a.log("will send e-mail alerts to %s", a.Server)

	return nil
}

func (a *SMTPAlerter) sendAlert(title, text string) error {
	var auth smtp.Auth

	if a.Username != "" && a.Password != "" {
		if a.CRMMD5 {
			auth = smtp.CRAMMD5Auth(a.Username, a.Password)
		} else {
			auth = smtp.PlainAuth(a.Identity, a.Username, a.Password, a.Server)
		}
	}

	msg := "Subject: " + title + "\r\n\r\n" + text

	var err error
	if !a.DisableTLS {
		err = smtp.SendMail(a.Server, auth, a.From, strings.Split(a.To, ","), []byte(msg))
	} else {
		c, err := smtp.Dial(a.Server)
		if err == nil {
			err = c.Mail(a.From)
			if err == nil {
				for _, addr := range strings.Split(a.To, ",") {
					if err = c.Rcpt(addr); err != nil {
						break
					}
				}
				if err == nil {
					wc, err := c.Data()
					if err == nil {
						_, err = wc.Write([]byte(msg))
						cerr := wc.Close()
						if err == nil {
							err = cerr
						}
						if err == nil {
							err = c.Quit()
						}
					}
				}
			}
		}
	}

	if err != nil {
		a.log("error sending e-mail %v with subject %v", err, title)
	}

	return err
}

func (a *SMTPAlerter) HandleProbeResult(p probe.Probe, d time.Duration, perr error) (err error) {
	a.m.Lock()
	defer a.m.Unlock()

	name := p.Name()

	st, has := a.status[name]
	if !has {
		st = &smtpServiceStatus{}
		a.status[name] = st
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
			err = a.sendAlert(newTitle, newDesc)
		}
	}

	st.inited = true

	return err
}
