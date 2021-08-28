package templates

import (
	"fmt"
	"github.com/Azure/Orkestra/pkg/graph"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EntrypointTemplateName  = "entry"
	HelmReleaseArg          = "helmrelease"
	TimeoutArg              = "timeout"
	HelmReleaseExecutorName = "helmrelease-executor"
	ChartMuseumName         = "chartmuseum"
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

func UpdateWorkflowTemplates(wf *v1alpha13.Workflow, tpls ...v1alpha13.Template) {
	wf.Spec.Templates = append(wf.Spec.Templates, tpls...)
}

func GenerateTemplates(graph *graph.Graph, namespace string, parallelism *int64) (*v1alpha13.Template, []v1alpha13.Template, error) {
	templateMap, err := generateAppDAGTemplates(graph, namespace, parallelism)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate application DAG templates : %w", err)
	}

	// Create the entry template from the app dag templates
	entryTemplate := &v1alpha13.Template{
		Name:        EntrypointTemplateName,
		DAG:         &v1alpha13.DAGTemplate{},
		Parallelism: parallelism,
	}

	var templateSlice []v1alpha13.Template
	for name, template := range templateMap {
		entryTemplate.DAG.Tasks = append(entryTemplate.DAG.Tasks, v1alpha13.DAGTask{
			Name:         utils.ConvertToDNS1123(template.Name),
			Template:     utils.ConvertToDNS1123(template.Name),
			Dependencies: utils.ConvertSliceToDNS1123(graph.Nodes[name].Dependencies),
		})
		templateSlice = append(templateSlice, template)
	}

	return entryTemplate, templateSlice, nil
}

func generateAppDAGTemplates(graph *graph.Graph, namespace string, parallelism *int64) (map[string]v1alpha13.Template, error) {
	templateMap := make(map[string]v1alpha13.Template, 0)

	for name, node := range graph.Nodes {
		template := v1alpha13.Template{
			Name:        utils.ConvertToDNS1123(node.Name),
			Parallelism: parallelism,
			DAG: &v1alpha13.DAGTemplate{
				Tasks: []v1alpha13.DAGTask{},
			},
		}
		for _, task := range node.Tasks {
			hr := createHelmRelease(task.Release, namespace, task.ChartName, task.ChartVersion)
			hr.Labels = map[string]string{
				v1alpha1.ChartLabel:     task.ChartName,
				v1alpha1.OwnershipLabel: graph.Name,
				v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
			}
			if task.Parent != "" {
				hr.Annotations = map[string]string{
					v1alpha1.ParentChartAnnotation: task.Parent,
				}
			}
			hrStr := utils.HrToB64AnyStringPtr(hr)
			for _, executor := range task.Executors {
				template.DAG.Tasks = append(template.DAG.Tasks, executor.GetTask(task.Name, task.Dependencies, getTimeout(task.Release.Timeout), hrStr))
			}
		}
		templateMap[name] = template
	}
	return templateMap, nil
}

func appDAGTaskBuilder(name string, dependencies []string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	task := v1alpha13.DAGTask{
		Name:     utils.ConvertToDNS1123(name),
		Template: HelmReleaseExecutorName,
		Arguments: v1alpha13.Arguments{
			Parameters: []v1alpha13.Parameter{
				{
					Name:  HelmReleaseArg,
					Value: hrStr,
				},
				{
					Name:  TimeoutArg,
					Value: timeout,
				},
			},
		},
		Dependencies: dependencies,
	}
	return task
}

func createHelmRelease(r *v1alpha1.Release, namespace, name, version string) *fluxhelmv2beta1.HelmRelease {
	return &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.ConvertToDNS1123(name),
			Namespace: r.TargetNamespace,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   utils.ConvertToDNS1123(name),
					Version: version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: namespace,
					},
				},
			},
			ReleaseName:     utils.ConvertToDNS1123(name),
			TargetNamespace: r.TargetNamespace,
			Timeout:         r.Timeout,
			Install:         r.Install,
			Upgrade:         r.Upgrade,
			Rollback:        r.Rollback,
			Uninstall:       r.Uninstall,
			Interval:        r.Interval,
			Values:          r.Values,
		},
	}
}
