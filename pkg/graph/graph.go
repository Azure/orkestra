package graph

import (
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	executorpkg "github.com/Azure/Orkestra/pkg/executor"
	"github.com/Azure/Orkestra/pkg/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	ValuesKeyGlobal = "global"
)

type Graph struct {
	Name         string
	AllExecutors map[string]executorpkg.Executor
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

func NewAppNode(application *v1alpha1.Application) *AppNode {
	return &AppNode{
		Name:         application.Name,
		Dependencies: application.Dependencies,
		Tasks:        make(map[string]*TaskNode),
	}
}

func (appNode *AppNode) DeepCopy() *AppNode {
	newAppNode := &AppNode{
		Name:         appNode.Name,
		Dependencies: appNode.Dependencies,
	}
	if appNode.Tasks != nil {
		newAppNode.Tasks = make(map[string]*TaskNode)
		for name, task := range appNode.Tasks {
			newAppNode.Tasks[name] = task.DeepCopy()
		}
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
	Executors    map[string]*ExecutorNode
}

func NewTaskNode(application *v1alpha1.Application) *TaskNode {
	return &TaskNode{
		Name:         getTaskName(application.Name, application.Name),
		ChartName:    application.Spec.Chart.Name,
		ChartVersion: application.Spec.Chart.Version,
		Release:      application.Spec.Release,
		Executors:    make(map[string]*ExecutorNode),
	}
}

func (taskNode *TaskNode) DeepCopy() *TaskNode {
	newTaskNode := &TaskNode{
		Name:         taskNode.Name,
		ChartName:    taskNode.ChartName,
		ChartVersion: taskNode.ChartVersion,
		Parent:       taskNode.Parent,
		Release:      taskNode.Release.DeepCopy(),
		Dependencies: taskNode.Dependencies,
	}
	if taskNode.Executors != nil {
		newTaskNode.Executors = make(map[string]*ExecutorNode)
		for name, executor := range taskNode.Executors {
			newTaskNode.Executors[name] = executor.DeepCopy()
		}
	}
	return newTaskNode
}

type ExecutorNode struct {
	Name         string
	Dependencies []string
	Executor     executorpkg.Executor
	Params       *apiextensionsv1.JSON
}

func NewExecutorNode(executor *v1alpha1.Executor) *ExecutorNode {
	return &ExecutorNode{
		Name:         executor.Name,
		Dependencies: executor.Dependencies,
		Executor:     executorpkg.ForwardFactory(executor.Type),
		Params:       executor.Params,
	}
}

func NewDefaultExecutorNode() *ExecutorNode {
	return &ExecutorNode{
		Name:     string(v1alpha1.HelmReleaseExecutor),
		Executor: executorpkg.ForwardFactory(v1alpha1.HelmReleaseExecutor),
	}
}

func (executorNode *ExecutorNode) DeepCopy() *ExecutorNode {
	return &ExecutorNode{
		Name:     executorNode.Name,
		Executor: executorNode.Executor,
		Params:   executorNode.Params,
	}
}

// NewForwardGraph takes in an ApplicationGroup and forms an abstracted
// DAG graph interface with app nodes and task nodes that can be passed into
// the template generation functions
func NewForwardGraph(appGroup *v1alpha1.ApplicationGroup) *Graph {
	g := &Graph{
		AllExecutors: make(map[string]executorpkg.Executor),
		Name:         appGroup.Name,
		Nodes:        make(map[string]*AppNode),
	}

	for i, application := range appGroup.Spec.Applications {
		applicationNode := NewAppNode(&application)
		applicationTaskNode := NewTaskNode(&application)
		g.assignExecutorsToTask(applicationTaskNode, application.Spec.Workflow)
		appValues := application.GetValues()

		// We need to know that the subcharts were staged in order to build this graph
		if len(appGroup.Status.Applications) > i {
			// Iterate through the subchart nodes
			for _, subChart := range application.Spec.Subcharts {
				subChartStatus, ok := appGroup.Status.Applications[i].Subcharts[subChart.Name]
				if !ok {
					continue
				}
				subChartVersion := subChartStatus.Version
				chartName := utils.GetSubchartName(application.Name, subChart.Name)

				// Get the sub-chart values and assign that ot the release
				values, _ := SubChartValues(subChart.Name, application.GetValues())
				release := application.Spec.Release.DeepCopy()
				release.Values = values

				subChartNode := &TaskNode{
					Name:         getTaskName(application.Name, subChart.Name),
					ChartName:    chartName,
					ChartVersion: subChartVersion,
					Release:      release,
					Parent:       application.Name,
					Executors:    make(map[string]*ExecutorNode),
				}
				for _, dep := range subChart.Dependencies {
					subChartNode.Dependencies = append(subChartNode.Dependencies, getTaskName(application.Name, dep))
				}

				g.assignExecutorsToTask(subChartNode, application.Spec.Workflow)
				applicationNode.Tasks[subChartNode.Name] = subChartNode

				// Disable the sub-chart dependencies in the values of the parent chart
				appValues[subChart.Name] = map[string]interface{}{
					"enabled": false,
				}

				// Add the node to the set of parent node dependencies
				applicationTaskNode.Dependencies = append(applicationTaskNode.Dependencies, subChartNode.Name)
			}
		}
		_ = applicationTaskNode.Release.SetValues(appValues)
		applicationNode.Tasks[applicationTaskNode.Name] = applicationTaskNode
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
	reverseGraph.clear()

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

			// Reverse the dependencies and the execution of the executors
			for _, executor := range subTask.Executors {
				for _, dep := range executor.Dependencies {
					if node, ok := subChartNode.Executors[dep]; ok {
						node.Dependencies = append(node.Dependencies, executor.Name)
					}
				}
				subChartNode.Executors[executor.Name].Executor = subChartNode.Executors[executor.Name].Executor.Reverse()
				reverseGraph.addExecutorIfNotExist(subChartNode.Executors[executor.Name].Executor)
			}
		}
	}
	return reverseGraph
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
	for name, node := range b.Nodes {
		if _, ok := a.Nodes[name]; !ok {
			combinedGraph.Nodes[name] = node.DeepCopy()
		}
	}
	for _, item := range b.AllExecutors {
		combinedGraph.addExecutorIfNotExist(item)
	}
	return combinedGraph
}

func (g *Graph) clear() *Graph {
	g.AllExecutors = make(map[string]executorpkg.Executor)
	for _, node := range g.Nodes {
		node.Dependencies = nil
		for _, task := range node.Tasks {
			task.Dependencies = nil
			for _, executor := range task.Executors {
				executor.Dependencies = nil
			}
		}
	}
	return g
}

func (g *Graph) addExecutorIfNotExist(executor executorpkg.Executor) {
	if _, ok := g.AllExecutors[executor.GetName()]; !ok {
		g.AllExecutors[executor.GetName()] = executor
	}
}

func getTaskName(appName, taskName string) string {
	return fmt.Sprintf("%s-%s", appName, taskName)
}

func (g *Graph) assignExecutorsToTask(taskNode *TaskNode, workflow []v1alpha1.Executor) {
	if len(workflow) == 0 {
		taskNode.Executors[string(v1alpha1.HelmReleaseExecutor)] = NewDefaultExecutorNode()
		g.addExecutorIfNotExist(executorpkg.ForwardFactory(v1alpha1.HelmReleaseExecutor))
	} else {
		for _, item := range workflow {
			taskNode.Executors[item.Name] = NewExecutorNode(&item)
			g.addExecutorIfNotExist(executorpkg.ForwardFactory(item.Type))
		}
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
