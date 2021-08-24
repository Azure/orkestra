package workflow

import (
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Graph struct {
	Name string
	Nodes map[string]*Node
}

type Node struct {
	Name string
	ChartName string
	ChartVersion string
	Owner string
	Release *v1alpha1.Release
	Dependencies []string
}

func NewForwardGraph(appGroup *v1alpha1.ApplicationGroup) *Graph {
	g := &Graph{
		Name: appGroup.Name,
		Nodes: make(map[string]*Node),
	}

	for i, application := range appGroup.Spec.Applications {
		applicationNode := NewNode(&application)
		appValues := applicationNode.Release.GetValues()

		// Iterate through the subchart nodes
		for _, subChart := range application.Spec.Subcharts {
			subChartVersion := appGroup.Status.Applications[i].Subcharts[subChart.Name].Version
			nodeName := utils.GetSubchartName(application.Name, subChart.Name)

			// Get the sub-chart values and assign that ot the release
			values, _ := subChartValues(subChart.Name, application.GetValues())
			release := application.Spec.Release.DeepCopy()
			release.Values = values

			subChartNode := &Node{
				Name: nodeName,
				ChartName: nodeName,
				ChartVersion: subChartVersion,
				Release: release,
				Owner: application.Name,
				Dependencies: []string{},
			}

			// Add the sub-chart dependencies with their names
			for _, dep := range subChart.Dependencies {
				subChartNode.Dependencies = append(subChartNode.Dependencies, utils.GetSubchartName(application.Name, dep))
			}
			subChartNode.Dependencies = append(subChartNode.Dependencies, application.Dependencies...)
			g.Nodes[nodeName] = subChartNode

			// Disable the sub-chart dependencies in the values of the parent chart
			appValues[subChart.Name] = map[string]interface{}{
				"enabled": false,
			}

			// Add the node to the set of parent node dependencies
			applicationNode.Dependencies = append(applicationNode.Dependencies, nodeName)
		}
		applicationNode.Release.SetValues(appValues)
		g.Nodes[applicationNode.Name] = applicationNode
	}
	return g
}

func NewReverseGraph(appGroup *v1alpha1.ApplicationGroup) *Graph {
	g := NewForwardGraph(appGroup).clearDependencies()

	for _, application := range appGroup.Spec.Applications {
		// Iterate through the application dependencies and reverse the dependency relationship
		for _, dep := range application.Dependencies {
			if node, ok := g.Nodes[dep]; ok {
				node.Dependencies = append(node.Dependencies, application.Name)
			}
		}
		for _, subChart := range application.Spec.Subcharts {
			nodeName := utils.GetSubchartName(application.Name, subChart.Name)
			subChartNode := g.Nodes[nodeName]

			// Application dependencies now depend on the sub-chart to reverse
			for _, dep := range application.Dependencies {
				if node, ok := g.Nodes[dep]; ok {
					node.Dependencies = append(node.Dependencies, subChartNode.Name)
				}
			}

			// Sub-chart dependencies now depend on this sub-chart to reverse
			for _, dep := range subChart.Dependencies {
				if node, ok := g.Nodes[utils.GetSubchartName(application.Name, dep)]; ok {
					node.Dependencies = append(node.Dependencies, subChartNode.Name)
				}
			}

			// Sub-chart now depends on the parent application chart to reverse
			subChartNode.Dependencies = append(subChartNode.Dependencies, application.Name)
		}
	}
	return g
}

func (g *Graph) clearDependencies() *Graph {
	for _, node := range g.Nodes {
		node.Dependencies = []string{}
	}
	return g
}

func NewNode(application *v1alpha1.Application) *Node {
	return &Node{
		Name: application.Name,
		ChartName: application.Spec.Chart.Name,
		ChartVersion: application.Spec.Chart.Version,
		Release: application.Spec.Release,
		Dependencies: application.Dependencies,
	}
}

func subChartValues(subChartName string, values map[string]interface{}) (*apiextensionsv1.JSON, error) {
	data := make(map[string]interface{})
	if scVals, ok := values[subChartName]; ok {
		if vv, ok := scVals.(map[string]interface{}); ok {
			for k, val := range vv {
				data[k] = val
			}
		}
	}
	if gVals, ok := values[ValuesKeyGlobal]; ok {
		if vv, ok := gVals.(map[string]interface{}); ok {
			data[ValuesKeyGlobal] = vv
		}
	}
	return v1alpha1.GetJSON(data)
}
