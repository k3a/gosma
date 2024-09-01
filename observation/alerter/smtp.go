package alerter

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/k3a/gosma/acquisition/probe"
	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/observation"
	"github.com/k3a/gosma/util"
	"gopkg.in/yaml.v3"
)

type SMTP struct {
	base

	Server     string `yaml:"server"`
	From       string `yaml:"from"`
	To         string `yaml:"to"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Identity   string `yaml:"identity"`
	CRMMD5     bool   `yaml:"crammd5"`
	DisableTLS bool   `yaml:"disable_tls"`
}

func (a *SMTP) err(estr string, args ...interface{}) error {
	return fmt.Errorf("smtp alerter: "+estr, args...)
}

func (a *SMTP) log(str string, args ...interface{}) {
	log.Printf("smtp alerter: "+str, args...)
}

func (a *SMTP) Init(n *yaml.Node) error {
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

func (a *SMTP) sendAlert(title, text string) error {
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

func (a *SMTP) ProbeStateChanged(p probe.Probe, st observation.ProbeStatus) (err error) {
	name := p.Name()

	title := ""
	desc := ""

	if st.CurrentState == "OK" {
		title = fmt.Sprintf("%s alert: %s OK", config.C.Name, name)
		desc = fmt.Sprintf("%s recovered after %s", name, time.Since(st.LastStateTransition).Round(time.Second))
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
			err = a.sendAlert(title, desc)
			if err != nil {
				log.Printf("smtp alerter: error %v", err)
			}
		}()
	}

	return nil
}
