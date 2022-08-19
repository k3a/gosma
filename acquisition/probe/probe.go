package probe

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type ResultReceiver interface {
	// ProbeResult is called by a probe in a new gorountine
	ProbeResult(p Probe, d time.Duration, err error)
}

// Probe lists operations which can be done on any probe
type Probe interface {
	fmt.Stringer
	Parent() Probe
	Name() string
	Type() string
	Description() string
	UpdateEvery() time.Duration
	AlertAfter() int
	ExecAfter() int
	ExecOnSuccess() string
	ExecOnFailure() string
	Children() Map
	RunLoop(stop <-chan struct{}, res ResultReceiver) error
}

// Map is a map of probes
type Map map[string]Probe

func (pm Map) String() string {
	s := ""
	for k, v := range pm {
		s += fmt.Sprintf("%s: %s\n", k, v)
	}
	return s[:len(s)-1]
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

func instantiateProbe(n *yaml.Node, bp *base) (Probe, error) {
	t := bp.Type()
	var p Probe

	switch t {
	case "http":
		p = &HTTP{base: *bp}
	case "file":
		p = &File{base: *bp}
	case "socket":
		p = &Socket{base: *bp}
	case "ftp":
		p = &FTP{base: *bp}
	case "dns":
		p = &DNS{base: *bp}
	case "smtp":
		p = &SMTP{base: *bp}
	case "imap":
		p = &IMAP{base: *bp}
	default:
		var dec interface{}
		n.Decode(&dec)
		return nil, fmt.Errorf("unknown type %s for %#v", t, dec)
	}

	err := n.Decode(p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// FromYAMLNode parses and instantiated a map of probes from a YAML MappingNode
func FromYAMLNode(parent *base, node *yaml.Node) (Map, error) {
	m := make(Map)

	key := true
	keyName := ""

	for _, n := range node.Content {
		if key {
			keyName = n.Value
		} else if n.Kind == yaml.MappingNode {
			bp, err := instantiateBaseProbe(parent, n, keyName)
			if err != nil {
				return nil, err
			}

			if isMappingNodeWithScalarValues(n) { // is leaf
				m[keyName], err = instantiateProbe(n, bp)
				if err != nil {
					return nil, err
				}
			} else { // is group
				bp.children, err = FromYAMLNode(bp, n)
				if err != nil {
					return nil, err
				}
				m[keyName] = bp
			}
		}

		key = !key
	}

	return m, nil
}
