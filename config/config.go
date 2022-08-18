package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

var Buildstamp = "DEBUG"

type config struct {
	Name     string      `yaml:"name"`
	URL      string      `yaml:"url"`
	Probes   yaml.Node   `yaml:"probes"`
	Watchers []yaml.Node `yaml:"watchers"`
}

func (c *config) validateAndSetDefaults() error {
	if c.Name == "" {
		c.Name = "GoSMA"
	}

	return nil
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

	return c.validateAndSetDefaults()
}

var C = &config{}
