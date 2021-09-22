package templates

import (
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/graph"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EntrypointTemplateName = "entry"
	ChartMuseumName        = "chartmuseum"
)

func GenerateWorkflow(name, namespace string, parallelism *int64) *v1alpha13.Workflow {
	return &v1alpha13.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{v1alpha1.HeritageLabel: v1alpha1.HeritageValue},
		},
		TypeMeta: v1.TypeMeta{
			APIVersion: v1alpha13.WorkflowSchemaGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha13.WorkflowSchemaGroupVersionKind.Kind,
		},
		Spec: v1alpha13.WorkflowSpec{
			Entrypoint:  EntrypointTemplateName,
			Templates:   make([]v1alpha13.Template, 0),
			Parallelism: parallelism,
			PodGC: &v1alpha13.PodGC{
				Strategy: v1alpha13.PodGCOnWorkflowCompletion,
			},
		},
	}
}

type TemplateGenerator struct {
	EntryTemplate v1alpha13.Template
	Templates     []v1alpha13.Template
	Namespace     string
	Parallelism   *int64
}

func NewTemplateGenerator(namespace string, parallelism *int64) *TemplateGenerator {
	return &TemplateGenerator{
		Namespace:   namespace,
		Parallelism: parallelism,
	}
}

func (tg *TemplateGenerator) AssignWorkflowTemplates(wf *v1alpha13.Workflow) {
	wf.Spec.Templates = append(wf.Spec.Templates, tg.Templates...)
	wf.Spec.Templates = append(wf.Spec.Templates, tg.EntryTemplate)
}

func (tg *TemplateGenerator) GenerateTemplates(graph *graph.Graph) error {
	tg.EntryTemplate = v1alpha13.Template{
		Name:        EntrypointTemplateName,
		DAG:         &v1alpha13.DAGTemplate{},
		Parallelism: tg.Parallelism,
	}

	// Create the entry template from the app dag templates
	for _, node := range graph.Nodes {
		template, err := tg.createNodeTemplate(node, graph.Name)
		if err != nil {
			return err
		}
		tg.Templates = append(tg.Templates, template)
		tg.EntryTemplate.DAG.Tasks = append(tg.EntryTemplate.DAG.Tasks, v1alpha13.DAGTask{
			Name:         template.Name,
			Template:     template.Name,
			Dependencies: utils.ConvertSliceToDNS1123(node.Dependencies),
		})
	}
	// Finally, add the executor templates in the graph to the set of templates
	tg.addExecutorTemplates(graph)
	return nil
}

func (tg *TemplateGenerator) createNodeTemplate(node *graph.AppNode, graphName string) (v1alpha13.Template, error) {
	template := v1alpha13.Template{
		Name:        utils.ConvertToDNS1123(node.Name),
		Parallelism: tg.Parallelism,
		DAG: &v1alpha13.DAGTemplate{
			Tasks: []v1alpha13.DAGTask{},
		},
	}
	for _, task := range node.Tasks {
		hrStr := utils.HrToB64(tg.createHelmRelease(task, graphName))
		if len(task.Executors) == 1 {
			// If we only have one executor, we don't need a sub-template
			// Just add this task to the application template
			for _, executorNode := range task.Executors {
				executorTask, err := executorNode.Executor.GetTask(task.Name, task.Dependencies, getTimeout(task.Release.Timeout), hrStr, executorNode.Params)
				if err != nil {
					return template, err
				}
				template.DAG.Tasks = append(template.DAG.Tasks, executorTask)
			}
		} else {
			// If we have more than one executor, we need to create the task
			// sub-template with executor dependencies
			taskTemplate, err := tg.createTaskTemplate(task, graphName)
			if err != nil {
				return template, err
			}
			tg.Templates = append(tg.Templates, taskTemplate)
			template.DAG.Tasks = append(template.DAG.Tasks, v1alpha13.DAGTask{
				Name:         taskTemplate.Name,
				Template:     taskTemplate.Name,
				Dependencies: task.Dependencies,
			})
		}
	}
	return template, nil
}

func (tg *TemplateGenerator) createTaskTemplate(task *graph.TaskNode, graphName string) (v1alpha13.Template, error) {
	hrStr := utils.HrToB64(tg.createHelmRelease(task, graphName))
	taskTemplate := v1alpha13.Template{
		Name:        utils.ConvertToDNS1123(task.Name),
		Parallelism: tg.Parallelism,
		DAG: &v1alpha13.DAGTemplate{
			Tasks: []v1alpha13.DAGTask{},
		},
	}
	for _, executorNode := range task.Executors {
		executorTask, err := executorNode.Executor.GetTask(executorNode.Name, executorNode.Dependencies, getTimeout(task.Release.Timeout), hrStr, executorNode.Params)
		if err != nil {
			return taskTemplate, err
		}
		taskTemplate.DAG.Tasks = append(taskTemplate.DAG.Tasks, executorTask)
	}
	return taskTemplate, nil
}

func (tg *TemplateGenerator) addExecutorTemplates(g *graph.Graph) {
	for _, executor := range g.AllExecutors {
		tg.Templates = append(tg.Templates, executor.GetTemplate())
	}
}

func (tg *TemplateGenerator) createHelmRelease(task *graph.TaskNode, graphName string) *fluxhelmv2beta1.HelmRelease {
	helmRelease := &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.ConvertToDNS1123(task.ChartName),
			Namespace: task.Release.TargetNamespace,
			Labels: map[string]string{
				v1alpha1.ChartLabel:     task.ChartName,
				v1alpha1.OwnershipLabel: graphName,
				v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
			},
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   utils.ConvertToDNS1123(task.ChartName),
					Version: task.ChartVersion,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: tg.Namespace,
					},
				},
			},
			ReleaseName:     utils.ConvertToDNS1123(task.ChartName),
			TargetNamespace: task.Release.TargetNamespace,
			Timeout:         task.Release.Timeout,
			Install:         task.Release.Install,
			Upgrade:         task.Release.Upgrade,
			Rollback:        task.Release.Rollback,
			Uninstall:       task.Release.Uninstall,
			Interval:        task.Release.Interval,
			Values:          task.Release.Values,
		},
	}
	if task.Parent != "" {
		helmRelease.Annotations = map[string]string{
			v1alpha1.ParentChartAnnotation: task.Parent,
		}
	}
	return helmRelease
}
