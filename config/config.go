package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var Buildstamp = "DEBUG"

const (
	Product    = "GoSMA"
	ProductURL = "https://github.com/k3a/gosma"
	Footer     = "<a href=\"" + ProductURL + "\" target=\"_blank\">" + Product + "</a> - Simple Monitoring and Alerting in Go"
)

type config struct {
	Name          string        `yaml:"name"`           // custom product/install name
	URL           string        `yaml:"url"`            // URL at which the installation responds
	Delay         time.Duration `yaml:"delay"`          // mute alers for this amount of time after startup
	RandomizePerc float64       `yaml:"randomize_perc"` // randomize update_for + or - this amount of percent
	Probes        yaml.Node     `yaml:"probes"`
	Watchers      []yaml.Node   `yaml:"watchers"`
	Alerters      []yaml.Node   `yaml:"alerters"`
}

func (c *config) LoadFromFile(cpath string) error {
	f, err := os.Open(cpath)
	if err != nil {
		return err
	}

	y := yaml.NewDecoder(f)

	err = y.Decode(&c)
	if err != nil {
		return err
	}

	if c.Name == "" {
		c.Name = Product
	}

	return nil
}

var C = &config{
	Name:          Product,
	RandomizePerc: 5,
}
