package workflow

import (
	"bytes"
	"encoding/base64"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	k8Yaml "k8s.io/apimachinery/pkg/util/yaml"
)

type Graph struct {
	nodes    map[string]v1alpha13.NodeStatus
	releases map[int][]fluxhelmv2beta1.HelmRelease
	maxLevel int
}

type Node struct {
	Status v1alpha13.NodeStatus
	Level  int
}

func Build(entry string, nodes map[string]v1alpha13.NodeStatus) (*Graph, error) {
	if len(nodes) == 0 {
		return nil, ErrNoNodesFound
	}

	g := &Graph{
		nodes:    nodes,
		releases: make(map[int][]fluxhelmv2beta1.HelmRelease),
	}

	e, ok := nodes[entry]
	if !ok {
		return nil, ErrEntryNodeNotFound
	}

	err := g.bft(e)
	if err != nil {
		return nil, err
	}

	return g, nil
}

// bft performs the Breath First Traversal of the DAG
func (g *Graph) bft(node v1alpha13.NodeStatus) error {
	visited := make(map[string]*Node)
	level := 0
	q := []v1alpha13.NodeStatus{}
	q = append(q, node)

	visited[node.ID] = &Node{
		Status: node,
		Level:  level,
	}

	for len(q) > 0 {
		level++
		n := q[0]
		for _, c := range n.Children {
			ch := g.nodes[c]
			if _, ok := visited[ch.ID]; !ok {
				// don't visit the child if it is reachable indirectly
				if !g.isIndirectChild(ch.ID, n) {
					// don't visit failed nodes
					if ch.Phase != v1alpha13.NodeSkipped &&
						ch.Phase != v1alpha13.NodeFailed &&
						ch.Phase != v1alpha13.NodeError {
						visited[ch.ID] = &Node{
							Status: ch,
							Level:  level,
						}
						q = append(q, ch)
					}
				}
			}
		}
		q = q[1:]
	}

	for _, v := range visited {
		if v.Status.Type != v1alpha13.NodeTypePod {
			continue
		}
		hrStr := v.Status.Inputs.Parameters[0].Value
		hrBytes, err := base64.StdEncoding.DecodeString(string(*hrStr))
		if err != nil {
			return err
		}
		hr := fluxhelmv2beta1.HelmRelease{}
		dec := k8Yaml.NewYAMLOrJSONDecoder(bytes.NewReader(hrBytes), 1000)
		if err := dec.Decode(&hr); err != nil {
			return err
		}

		if _, ok := g.releases[v.Level]; !ok {
			g.releases[v.Level] = make([]fluxhelmv2beta1.HelmRelease, 0)
		}
		g.releases[v.Level] = append(g.releases[v.Level], hr)
	}

	g.maxLevel = level
	return nil
}

func (g *Graph) isIndirectChild(nodeID string, node v1alpha13.NodeStatus) bool {
	for _, c := range node.Children {
		ch := g.nodes[c]
		if ch.ID != nodeID && g.isChild(nodeID, ch) {
			return true
		}
	}

	return false
}

func (g *Graph) isChild(nodeID string, node v1alpha13.NodeStatus) bool {
	visited := make(map[string]bool)
	q := []v1alpha13.NodeStatus{}
	q = append(q, node)

	visited[node.ID] = true

	for len(q) > 0 {
		n := q[0]
		for _, c := range n.Children {
			ch := g.nodes[c]
			if ch.ID == nodeID {
				return true
			}
			if !visited[ch.ID] {
				visited[ch.ID] = true
				q = append(q, ch)
			}
		}
		q = q[1:]
	}

	return false
}

func (g *Graph) Reverse() [][]fluxhelmv2beta1.HelmRelease {
	reverseSlice := make([][]fluxhelmv2beta1.HelmRelease, 0)
	for i := g.maxLevel; i >= 0; i-- {
		if _, ok := g.releases[i]; ok {
			reverseSlice = append(reverseSlice, g.releases[i])
		}
	}
	return reverseSlice
}
