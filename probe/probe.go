package probe

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type ProbeResultReceiver interface {
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
	Children() ProbeMap
	RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error
}

// ProbeMap is a map of probes
type ProbeMap map[string]Probe

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

// ParseProbesFromYAMLNode parses and instantiated a map of probes from a YAML MappingNode
func ParseProbesFromYAMLNode(parent *BaseProbe, node *yaml.Node) (ProbeMap, error) {
	m := make(ProbeMap)

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
				bp.children, err = ParseProbesFromYAMLNode(bp, n)
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
