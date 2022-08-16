package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type ProbeMap map[string]Probe

func (pm ProbeMap) String() string {
	s := ""
	for k, v := range pm {
		s += fmt.Sprintf("%s: %s\n", k, v)
	}
	return s[:len(s)-1]
}

type Probe interface {
	fmt.Stringer
	Parent() Probe
	Name() string
	Type() string
	Description() string
	UpdateEvery() time.Duration
	Children() ProbeMap
	RunLoop(c *Config, ch <-chan struct{}) error
}

type BaseProbe struct {
	name   string
	parent Probe

	YType        string        `yaml:"type"`
	YDescription string        `yaml:"description"`
	YUpdateEvery time.Duration `yaml:"update_every"`
	children     ProbeMap      `yaml:"-"`
}

func (bp *BaseProbe) RunLoop(c *Config, ch <-chan struct{}) error {
	return nil
}

func (bp *BaseProbe) Name() string {
	t := bp.name

	p := bp.Parent()
	for !reflect.ValueOf(p).IsNil() {
		t = p.Name() + "." + t
		p = p.Parent()
	}

	return t
}

func (bp *BaseProbe) Parent() Probe {
	return bp.parent
}

func (bp *BaseProbe) Type() string {
	t := bp.YType

	p := bp.Parent()
	for t == "" && !reflect.ValueOf(p).IsNil() {
		t = p.Type()
		p = p.Parent()
	}

	bp.YType = t
	return t
}

func (bp *BaseProbe) Description() string {
	t := bp.YDescription

	p := bp.Parent()
	for t == "" && !reflect.ValueOf(p).IsNil() {
		t = p.Description()
		p = p.Parent()
	}

	bp.YDescription = t
	return t
}

func (bp *BaseProbe) UpdateEvery() time.Duration {
	d := bp.YUpdateEvery

	p := bp.Parent()
	for d == 0 && !reflect.ValueOf(p).IsNil() {
		d = p.UpdateEvery()
		p = p.Parent()
	}

	if d == 0 {
		d = 30 * time.Second // default update interval
	}

	bp.YUpdateEvery = d
	return d
}

func (bp *BaseProbe) Children() ProbeMap {
	return bp.children
}

func (bp *BaseProbe) String() string {
	if bp == nil {
		return "<nil>"
	}

	indent := ""
	p := bp.Parent()
	for !reflect.ValueOf(p).IsNil() {
		indent += " "
		p = p.Parent()
	}

	s := fmt.Sprintf("BaseProbe with %d children:\n", len(bp.children))

	for k, v := range bp.children {
		s += fmt.Sprintf("%s %s: %s\n", indent, k, v.String())
	}

	return s[:len(s)-1]
}

type HTTPProbe struct {
	BaseProbe
	URL           string `yaml:"url"`
	ResponseMatch string `yaml:"response_match"`
}

func (p *HTTPProbe) RunLoop(c *Config, stop <-chan struct{}) error {
	// validate
	if p.URL == "" {
		return fmt.Errorf("HTTPProbe: url is required")
	}

loop:
	for {
		select {
		case <-stop:
			break loop
		case <-time.After(p.UpdateEvery()):
			log.Printf("%s: works", p.Name())
		}
	}

	return nil
}

func (p *HTTPProbe) String() string {
	attrs := "url=" + p.URL
	if p.ResponseMatch != "" {
		attrs += ", response_match=" + p.ResponseMatch
	}
	attrs += ", name=" + p.Name()
	return fmt.Sprintf("HTTPProbe(%s)", attrs)
}

type Config struct {
	YProbes yaml.Node `yaml:"probes"`
	Probes  ProbeMap  `yaml:"-"`
}

func errChk(err error, desc string) {
	if err != nil {
		fmt.Fprint(os.Stderr, "ERROR "+desc+": "+err.Error()+"\n")
		os.Exit(9)
	}
}

func isMappingNodeWithScalarValues(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}

	key := true
	for _, n := range node.Content {
		if !key && n.Kind == yaml.MappingNode {
			return false
		}

		key = !key
	}

	return true
}

func instantiateProbe(n *yaml.Node, bp *BaseProbe) (Probe, error) {
	t := bp.Type()
	switch t {
	case "http":
		var hp = HTTPProbe{BaseProbe: *bp}
		err := n.Decode(&hp)
		if err != nil {
			return nil, err
		}
		return &hp, nil
	default:
		var dec interface{}
		n.Decode(&dec)
		return nil, fmt.Errorf("unknown type %s for %#v", t, dec)
	}
}

func parseProbeNode(parent *BaseProbe, node *yaml.Node) (ProbeMap, error) {
	m := make(ProbeMap)

	key := true
	keyName := ""

	for _, n := range node.Content {
		if key {
			keyName = n.Value
		} else if n.Kind == yaml.MappingNode {
			var bp = BaseProbe{parent: parent, name: keyName}
			err := n.Decode(&bp)
			if err != nil {
				return nil, err
			}

			if isMappingNodeWithScalarValues(n) { // is leaf
				m[keyName], err = instantiateProbe(n, &bp)
				if err != nil {
					return nil, err
				}
			} else { // is group
				bp.children, err = parseProbeNode(&bp, n)
				if err != nil {
					return nil, err
				}
				m[keyName] = &bp
			}
		}

		key = !key
	}

	return m, nil
}

func runProbeLoopInBackground(p Probe, c *Config, ch <-chan struct{}) {
	go func() {
		err := p.RunLoop(c, ch)
		if err != nil {
			log.Printf("error running %s: %v\n", p.Name(), err)
		}
	}()

	for _, cp := range p.Children() {
		runProbeLoopInBackground(cp, c, ch)
	}
}

func main() {
	f, err := os.Open("testdata/ndemic.yaml")
	errChk(err, "opening file")

	y := yaml.NewDecoder(f)

	c := Config{}

	err = y.Decode(&c)
	errChk(err, "decoding config file")

	c.Probes, err = parseProbeNode(nil, &c.YProbes)
	errChk(err, "parsing probe nodes")

	// print all probes
	fmt.Println(c.Probes)

	// start all probes
	ch := make(chan struct{})
	for _, p := range c.Probes {
		runProbeLoopInBackground(p, &c, ch)
	}

	// to stop all
	// time.Sleep(3 * time.Second)
	// close(ch)

	err = http.ListenAndServe("0.0.0.0:1840", nil)
	errChk(err, "listening for http traffic")
}
