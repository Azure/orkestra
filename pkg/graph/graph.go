package graph

import (
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/executor"
	"github.com/Azure/Orkestra/pkg/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	ValuesKeyGlobal = "global"
)

type Graph struct {
	Name         string
	AllExecutors []executor.Executor
	Nodes        map[string]*AppNode
}

func (g *Graph) DeepCopy() *Graph {
	newGraph := &Graph{
		Name:         g.Name,
		Nodes:        make(map[string]*AppNode),
		AllExecutors: g.AllExecutors,
	}
	for name, appNode := range g.Nodes {
		newGraph.Nodes[name] = appNode.DeepCopy()
	}
	return newGraph
}

type AppNode struct {
	Name         string
	Dependencies []string
	Tasks        map[string]*TaskNode
}

func (appNode *AppNode) DeepCopy() *AppNode {
	newAppNode := &AppNode{
		Name:         appNode.Name,
		Dependencies: appNode.Dependencies,
		Tasks:        make(map[string]*TaskNode),
	}

	for name, task := range appNode.Tasks {
		newAppNode.Tasks[name] = task.DeepCopy()
	}
	return newAppNode
}

type TaskNode struct {
	Name         string
	ChartName    string
	ChartVersion string
	Parent       string
	Release      *v1alpha1.Release
	Dependencies []string
	Executors    []executor.Executor
}

func (taskNode *TaskNode) DeepCopy() *TaskNode {
	return &TaskNode{
		Name:         taskNode.Name,
		ChartName:    taskNode.ChartName,
		ChartVersion: taskNode.ChartVersion,
		Parent:       taskNode.Parent,
		Release:      taskNode.Release.DeepCopy(),
		Dependencies: taskNode.Dependencies,
		Executors:    taskNode.Executors,
	}
}

// NewForwardGraph takes in an ApplicationGroup and forms an abstracted
// DAG graph interface with app nodes and task nodes that can be passed into
// the template generation functions
func NewForwardGraph(appGroup *v1alpha1.ApplicationGroup) *Graph {
	g := &Graph{
		Name:         appGroup.Name,
		Nodes:        make(map[string]*AppNode),
		AllExecutors: []executor.Executor{executor.DefaultForward{}},
	}

	for i, application := range appGroup.Spec.Applications {
		applicationNode := NewAppNode(&application)
		applicationNode.Tasks[application.Name] = NewTaskNode(&application)
		appValues := application.Spec.Release.GetValues()

		// We need to know that the subcharts were staged in order to build this graph
		if len(appGroup.Status.Applications) > i {
			// Iterate through the subchart nodes
			for _, subChart := range application.Spec.Subcharts {
				if data, ok := appGroup.Status.Applications[i].Subcharts[subChart.Name]; ok {
					subChartVersion := data.Version
					chartName := utils.GetSubchartName(application.Name, subChart.Name)

					// Get the sub-chart values and assign that ot the release
					values, _ := SubChartValues(subChart.Name, application.GetValues())
					release := application.Spec.Release.DeepCopy()
					release.Values = values

					subChartNode := &TaskNode{
						Name:         subChart.Name,
						ChartName:    chartName,
						ChartVersion: subChartVersion,
						Release:      release,
						Parent:       application.Name,
						Dependencies: subChart.Dependencies,
						Executors:    []executor.Executor{executor.DefaultForward{}},
					}

					applicationNode.Tasks[subChart.Name] = subChartNode

					// Disable the sub-chart dependencies in the values of the parent chart
					appValues[subChart.Name] = map[string]interface{}{
						"enabled": false,
					}

					// Add the node to the set of parent node dependencies
					applicationNode.Tasks[application.Name].Dependencies = append(applicationNode.Tasks[application.Name].Dependencies, subChart.Name)
				}
			}
		}
		_ = applicationNode.Tasks[application.Name].Release.SetValues(appValues)

		g.Nodes[applicationNode.Name] = applicationNode
	}
	return g
}

// NewReverseGraph creates a new reversed DAG based off the passed ApplicationGroup
func NewReverseGraph(appGroup *v1alpha1.ApplicationGroup) *Graph {
	return NewForwardGraph(appGroup).Reverse()
}

// Reverse method reverses the app node dependencies and the task node dependencies
// of the received graph
func (g *Graph) Reverse() *Graph {
	// DeepCopy so that we can clear dependencies
	reverseGraph := g.DeepCopy()
	reverseGraph.AssignExecutors(executor.DefaultReverse{})
	reverseGraph.clearDependencies()

	for _, application := range g.Nodes {
		// Iterate through the application dependencies and reverse the dependency relationship
		for _, dep := range application.Dependencies {
			if node, ok := reverseGraph.Nodes[dep]; ok {
				node.Dependencies = append(node.Dependencies, application.Name)
			}
		}
		for _, subTask := range application.Tasks {
			subChartNode := reverseGraph.Nodes[application.Name].Tasks[subTask.Name]
			// Sub-chart dependencies now depend on this sub-chart to reverse
			for _, dep := range subTask.Dependencies {
				if node, ok := reverseGraph.Nodes[application.Name].Tasks[dep]; ok {
					node.Dependencies = append(node.Dependencies, subChartNode.Name)
				}
			}
		}
	}
	return reverseGraph
}

func (g *Graph) AssignExecutors(executors ...executor.Executor) *Graph {
	g.AllExecutors = executors
	for _, appNode := range g.Nodes {
		for _, taskNode := range appNode.Tasks {
			taskNode.Executors = executors
		}
	}
	return g
}

// Diff returns the difference between two graphs
// It is the equivalent of performing A - B
func Diff(a, b *Graph) *Graph {
	diffGraph := a.DeepCopy()
	for name, appA := range a.Nodes {
		if appB, ok := b.Nodes[name]; ok {
			gotAllTasks := true
			for taskName := range appA.Tasks {
				if _, ok := appB.Tasks[taskName]; ok {
					delete(diffGraph.Nodes[name].Tasks, taskName)
				} else {
					gotAllTasks = false
				}
			}
			if gotAllTasks {
				delete(diffGraph.Nodes, name)
			}
		}
	}
	return diffGraph
}

// Combine adds app nodes from the second graph to the first graph
// If an app node with the same name exists in second graph from the first
// graph, it is ignored.
func Combine(a, b *Graph) *Graph {
	combinedGraph := a.DeepCopy()
	combinedGraph.AllExecutors = append(combinedGraph.AllExecutors, b.AllExecutors...)
	for name, node := range b.Nodes {
		if _, ok := a.Nodes[name]; !ok {
			combinedGraph.Nodes[name] = node.DeepCopy()
		}
	}
	return combinedGraph
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
		Executors:    []executor.Executor{executor.DefaultForward{}},
	}
}

// SubChartValues is the equivalent function to what helm client does with the global
// values file and its subchart values
func SubChartValues(subChartName string, values map[string]interface{}) (*apiextensionsv1.JSON, error) {
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
