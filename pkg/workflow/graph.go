package workflow

import (
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Graph struct {
	Name  string
	Nodes map[string]*AppNode
}

type AppNode struct {
	Name         string
	Dependencies []string
	Tasks        map[string]*TaskNode
}

type TaskNode struct {
	Name         string
	ChartName    string
	ChartVersion string
	Parent       string
	Release      *v1alpha1.Release
	Dependencies []string
}

func NewForwardGraph(appGroup *v1alpha1.ApplicationGroup) *Graph {
	g := &Graph{
		Name:  appGroup.Name,
		Nodes: make(map[string]*AppNode),
	}

	for i, application := range appGroup.Spec.Applications {
		applicationNode := NewAppNode(&application)
		applicationNode.Tasks[application.Name] = NewTaskNode(&application)
		appValues := application.Spec.Release.GetValues()

		// Iterate through the subchart nodes
		for _, subChart := range application.Spec.Subcharts {
			subChartVersion := appGroup.Status.Applications[i].Subcharts[subChart.Name].Version
			chartName := utils.GetSubchartName(application.Name, subChart.Name)

			// Get the sub-chart values and assign that ot the release
			values, _ := subChartValues(subChart.Name, application.GetValues())
			release := application.Spec.Release.DeepCopy()
			release.Values = values

			subChartNode := &TaskNode{
				Name:         subChart.Name,
				ChartName:    chartName,
				ChartVersion: subChartVersion,
				Release:      release,
				Parent:       application.Name,
				Dependencies: subChart.Dependencies,
			}

			applicationNode.Tasks[subChart.Name] = subChartNode

			// Disable the sub-chart dependencies in the values of the parent chart
			appValues[subChart.Name] = map[string]interface{}{
				"enabled": false,
			}

			// Add the node to the set of parent node dependencies
			applicationNode.Tasks[application.Name].Dependencies = append(applicationNode.Tasks[application.Name].Dependencies, subChart.Name)
		}
		_ = applicationNode.Tasks[application.Name].Release.SetValues(appValues)

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
			subChartNode := g.Nodes[application.Name].Tasks[subChart.Name]

			// Sub-chart dependencies now depend on this sub-chart to reverse
			for _, dep := range subChart.Dependencies {
				if node, ok := g.Nodes[application.Name].Tasks[dep]; ok {
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
		node.Dependencies = nil
		for _, task := range node.Tasks {
			task.Dependencies = nil
		}
	}
	return g
}

func NewAppNode(application *v1alpha1.Application) *AppNode {
	return &AppNode{
		Name:         application.Name,
		Dependencies: application.Dependencies,
		Tasks:        make(map[string]*TaskNode),
	}
}

func NewTaskNode(application *v1alpha1.Application) *TaskNode {
	return &TaskNode{
		Name:         application.Name,
		ChartName:    application.Spec.Chart.Name,
		ChartVersion: application.Spec.Chart.Version,
		Release:      application.Spec.Release,
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
		if vv, ok := scVals.(map[string]string); ok {
			for k, val := range vv {
				data[k] = val
			}
		}
	}
	if gVals, ok := values[ValuesKeyGlobal]; ok {
		if vv, ok := gVals.(map[string]interface{}); ok {
			data[ValuesKeyGlobal] = vv
		}
		if vv, ok := gVals.(map[string]string); ok {
			data[ValuesKeyGlobal] = vv
		}
	}
	return v1alpha1.GetJSON(data)
}
