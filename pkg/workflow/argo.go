package workflow

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	argoAPIVersion    = "argoproj.io/v1alpha1"
	argoKind          = "Workflow"
	entrypointTplName = "entry"

	helmReleaseArg             = "helmrelease"
	timeoutArg                 = "timeout"
	helmReleaseExecutor        = "helmrelease-executor"
	helmReleaseReverseExecutor = "helmrelease-reverse-executor"

	valuesKeyGlobal = "global"
	ChartLabelKey   = "chart"
)

var (
	defaultTimeout = "5m"
)

type argo struct {
	scheme *runtime.Scheme
	cli    client.Client
	wf     *v1alpha12.Workflow
	rwf    *v1alpha12.Workflow

	stagingRepoURL string
	parallelism    *int64
}

// Argo implements the Workflow interface for the Argo Workflow based DAG engine
func Argo(scheme *runtime.Scheme, c client.Client, stagingRepoURL string, workflowParallelism int64) *argo { //nolint:golint
	return &argo{
		scheme:         scheme,
		cli:            c,
		stagingRepoURL: stagingRepoURL,
		parallelism:    &workflowParallelism,
	}
}

func (a *argo) initWorkflowObject() *v1alpha12.Workflow {
	return &v1alpha12.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels: map[string]string{HeritageLabel: Project},
		},
		TypeMeta: v1.TypeMeta{
			APIVersion: argoAPIVersion,
			Kind:       argoKind,
		},
		Spec: v1alpha12.WorkflowSpec{
			Entrypoint:  entrypointTplName,
			Templates:   make([]v1alpha12.Template, 0),
			Parallelism: a.parallelism,
			PodGC: &v1alpha12.PodGC{
				Strategy: v1alpha12.PodGCOnWorkflowCompletion,
			},
		},
	}
}

func (a *argo) Generate(ctx context.Context, l logr.Logger, g *v1alpha1.ApplicationGroup) error {
	if g == nil {
		l.Error(nil, "ApplicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	a.wf = a.initWorkflowObject()

	// Set name and namespace based on the input application group
	a.wf.Name = g.Name
	a.wf.Namespace = workflowNamespace()

	err := a.generateWorkflow(ctx, g)
	if err != nil {
		l.Error(err, "failed to generate workflow")
		return fmt.Errorf("failed to generate argo workflow : %w", err)
	}

	return nil
}

func (a *argo) Submit(ctx context.Context, l logr.Logger, g *v1alpha1.ApplicationGroup) error {
	if a.wf == nil {
		l.Error(nil, "workflow object cannot be nil")
		return fmt.Errorf("workflow object cannot be nil")
	}

	if g == nil {
		l.Error(nil, "applicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	obj := &v1alpha12.Workflow{
		ObjectMeta: v1.ObjectMeta{Labels: make(map[string]string)},
	}

	namespaces := []string{}
	for _, app := range g.Spec.Applications {
		namespaces = append(namespaces, app.Spec.Release.TargetNamespace)
	}

	for _, namespace := range namespaces {
		ns := corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: namespace,
			},
		}
		// Create the namespace since helm-operator does not do this
		err := a.cli.Get(ctx, types.NamespacedName{Name: ns.Name}, &ns)
		if err != nil {
			if errors.IsNotFound(err) {
				// Add OwnershipReference
				err = controllerutil.SetControllerReference(g, &ns, a.scheme)
				if err != nil {
					return fmt.Errorf("failed to set OwnerReference for Namespace %s : %w", ns.Name, err)
				}

				err = a.cli.Create(ctx, &ns)
				if err != nil {
					return fmt.Errorf("failed to CREATE namespace %s object : %w", ns.Name, err)
				}
			}
		}
	}

	err := a.cli.Get(ctx, types.NamespacedName{Namespace: a.wf.Namespace, Name: a.wf.Name}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Add OwnershipReference
			err = controllerutil.SetControllerReference(g, a.wf, a.scheme)
			if err != nil {
				l.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
				return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
			}

			a.wf.Labels[OwnershipLabel] = g.Name

			// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
			// create a new object and submit it to the cluster
			err = a.cli.Create(ctx, a.wf)
			if err != nil {
				l.Error(err, "failed to CREATE argo workflow object")
				return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
			}
		} else {
			l.Error(err, "failed to GET workflow object with an unrecoverable error")
			return fmt.Errorf("failed to GET workflow object with an unrecoverable error : %w", err)
		}
	}

	// If the workflow needs an update, delete the previous workflow and apply the new one
	// Argo Workflow does not rerun the workflow on UPDATE, so intead we cleanup and reapply
	if g.Status.Update {
		err = a.cli.Delete(ctx, obj)
		if err != nil {
			l.Error(err, "failed to DELETE argo workflow object")
			return fmt.Errorf("failed to DELETE argo workflow object : %w", err)
		}
		// If the argo Workflow object is Found on the cluster
		// update the workflow and submit it to the cluster
		// Add OwnershipReference
		err = controllerutil.SetControllerReference(g, a.wf, a.scheme)
		if err != nil {
			l.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
			return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
		}

		a.wf.Labels[OwnershipLabel] = g.Name

		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		err = a.cli.Create(ctx, a.wf)
		if err != nil {
			l.Error(err, "failed to CREATE argo workflow object")
			return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
		}

		g.Status.Update = false
	}

	return nil
}

func (a *argo) GenerateReverse(ctx context.Context, l logr.Logger, nodes map[string]v1alpha12.NodeStatus, wf *v1alpha12.Workflow) error {
	if wf == nil {
		l.Error(nil, "forward workflow object cannot be nil")
		return fmt.Errorf("forward workflow object cannot be nil")
	}

	a.rwf = a.initWorkflowObject()

	// Set name and namespace based on the forward workflow
	a.rwf.Name = fmt.Sprintf("%s-reverse", wf.Name)
	a.rwf.Namespace = workflowNamespace()

	err := a.generateReverseWorkflow(ctx, l, nodes, wf)
	if err != nil {
		l.Error(err, "failed to generate reverse workflow")
		return fmt.Errorf("failed to generate argo reverse workflow : %w", err)
	}

	return nil
}

func (a *argo) SubmitReverse(ctx context.Context, l logr.Logger, wf *v1alpha12.Workflow) error {
	if a.rwf == nil {
		l.Error(nil, "reverse workflow object cannot be nil")
		return fmt.Errorf("reverse workflow object cannot be nil")
	}

	if wf == nil {
		l.Error(nil, "forward workflow object cannot be nil")
		return fmt.Errorf("forward workflow object cannot be nil")
	}

	obj := &v1alpha12.Workflow{}

	err := a.cli.Get(ctx, types.NamespacedName{Namespace: a.rwf.Namespace, Name: a.rwf.Name}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Add OwnershipReference
			err = controllerutil.SetControllerReference(wf, a.rwf, a.scheme)
			if err != nil {
				l.Error(err, "unable to set forward workflow as owner of Argo reverse Workflow object")
				return fmt.Errorf("unable to set forward workflow as owner of Argo reverse Workflow: %w", err)
			}

			// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
			// create a new object and submit it to the cluster
			err = a.cli.Create(ctx, a.rwf)
			if err != nil {
				l.Error(err, "failed to CREATE argo workflow object")
				return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
			}
		} else {
			l.Error(err, "failed to GET workflow object with an unrecoverable error")
			return fmt.Errorf("failed to GET workflow object with an unrecoverable error : %w", err)
		}
	}

	return nil
}

func (a *argo) generateWorkflow(ctx context.Context, g *v1alpha1.ApplicationGroup) error {
	// Generate the Entrypoint template and Application Group DAG
	err := a.generateAppGroupTpls(ctx, g)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}

	return nil
}

func getTaskNamesFromHelmReleases(bucket []fluxhelmv2beta1.HelmRelease) []string {
	out := []string{}
	for _, hr := range bucket {
		out = append(out, pkg.ConvertToDNS1123(hr.GetReleaseName()+"-"+(hr.Namespace)))
	}
	return out
}

func (a *argo) generateReverseWorkflow(ctx context.Context, l logr.Logger, nodes map[string]v1alpha12.NodeStatus, wf *v1alpha12.Workflow) error {
	graph, err := Build(wf.Name, nodes)
	if err != nil {
		l.Error(err, "failed to build the wf status DAG")
		return fmt.Errorf("failed to build the wf status DAG : %w", err)
	}

	rev := graph.Reverse()

	entry := v1alpha12.Template{
		Name: entrypointTplName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, 0),
		},
	}

	var prevbucket []fluxhelmv2beta1.HelmRelease
	for _, bucket := range rev {
		for _, hr := range bucket {
			task := v1alpha12.DAGTask{
				Name:     pkg.ConvertToDNS1123(hr.GetReleaseName() + "-" + hr.Namespace),
				Template: helmReleaseReverseExecutor,
				Arguments: v1alpha12.Arguments{
					Parameters: []v1alpha12.Parameter{
						{
							Name:  helmReleaseArg,
							Value: strToStrPtr(hrToYAML(hr)),
						},
					},
				},
				Dependencies: convertSliceToDNS1123(getTaskNamesFromHelmReleases(prevbucket)),
			}

			entry.DAG.Tasks = append(entry.DAG.Tasks, task)
		}
		prevbucket = bucket
	}

	if len(entry.DAG.Tasks) == 0 {
		return fmt.Errorf("entry template must have at least one task")
	}

	updateWorkflowTemplates(a.rwf, entry)

	updateWorkflowTemplates(a.rwf, defaultReverseExecutor())

	return nil
}

func (a *argo) generateAppGroupTpls(ctx context.Context, g *v1alpha1.ApplicationGroup) error {
	if a.wf == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	if g == nil {
		return fmt.Errorf("applicationGroup cannot be nil")
	}

	entry := v1alpha12.Template{
		Name: entrypointTplName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, len(g.Spec.Applications)),
			// TBD (nitishm): Do we need to failfast?
			// FailFast: true
		},
		Parallelism: a.parallelism,
	}

	adt, err := a.generateAppDAGTemplates(ctx, g, a.stagingRepoURL)
	if err != nil {
		return fmt.Errorf("failed to generate application DAG templates : %w", err)
	}
	updateWorkflowTemplates(a.wf, adt...)

	err = updateAppGroupDAG(g, &entry, adt)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}
	updateWorkflowTemplates(a.wf, entry)

	// TODO: Add the executor template
	// This should eventually be configurable
	updateWorkflowTemplates(a.wf, defaultExecutor())

	return nil
}

func updateAppGroupDAG(g *v1alpha1.ApplicationGroup, entry *v1alpha12.Template, tpls []v1alpha12.Template) error {
	if entry == nil {
		return fmt.Errorf("entry template cannot be nil")
	}

	entry.DAG = &v1alpha12.DAGTemplate{
		Tasks: make([]v1alpha12.DAGTask, len(tpls), len(tpls)),
	}

	for i, tpl := range tpls {
		entry.DAG.Tasks[i] = v1alpha12.DAGTask{
			Name:         pkg.ConvertToDNS1123(tpl.Name),
			Template:     pkg.ConvertToDNS1123(tpl.Name),
			Dependencies: convertSliceToDNS1123(g.Spec.Applications[i].Dependencies),
		}
	}

	return nil
}

func (a *argo) generateAppDAGTemplates(ctx context.Context, g *v1alpha1.ApplicationGroup, repo string) ([]v1alpha12.Template, error) {
	ts := make([]v1alpha12.Template, 0)

	for i, app := range g.Spec.Applications {
		var hasSubcharts bool
		appStatus := &g.Status.Applications[i].ChartStatus
		scStatus := g.Status.Applications[i].Subcharts

		// Create Subchart DAG only when the application chart has dependencies
		if len(app.Spec.Subcharts) > 0 {
			hasSubcharts = true
			t := v1alpha12.Template{
				Name:        pkg.ConvertToDNS1123(app.Name),
				Parallelism: a.parallelism,
			}

			t.DAG = &v1alpha12.DAGTemplate{}
			tasks, err := a.generateSubchartAndAppDAGTasks(ctx, g, &app, appStatus, scStatus, repo, app.Spec.Release.TargetNamespace)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Application Template DAG tasks : %w", err)
			}

			t.DAG.Tasks = tasks

			ts = append(ts, t)
		}

		if !hasSubcharts {
			hr := fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      pkg.ConvertToDNS1123(app.Name),
					Namespace: app.Spec.Release.TargetNamespace,
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   app.Spec.Chart.Name,
							Version: app.Spec.Chart.Version,
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: workflowNamespace(),
							},
						},
					},
					Interval:        app.Spec.Release.Interval,
					ReleaseName:     pkg.ConvertToDNS1123(app.Name),
					TargetNamespace: app.Spec.Release.TargetNamespace,
					Timeout:         app.Spec.Release.Timeout,
					Values:          app.Spec.Release.Values,
				},
			}
			if app.Spec.Release.Install != nil {
				hr.Spec.Install = &fluxhelmv2beta1.Install{
					DisableWait: app.Spec.Release.DisableWaitForInstall(),
				}
			}
			if app.Spec.Release.Upgrade != nil {
				hr.Spec.Upgrade = &fluxhelmv2beta1.Upgrade{
					DisableWait: app.Spec.Release.DisableWaitForUpgrade(),
					Force:       app.Spec.Release.ForceForUpgrade(),
				}
			}
			if app.Spec.Release.Rollback != nil {
				hr.Spec.Rollback = &fluxhelmv2beta1.Rollback{
					DisableWait: app.Spec.Release.DisableWaitForRollback(),
				}
			}
			hr.Labels = map[string]string{
				ChartLabelKey:  app.Name,
				OwnershipLabel: g.Name,
				HeritageLabel:  Project,
			}

			tApp := v1alpha12.Template{
				Name:        pkg.ConvertToDNS1123(app.Name),
				Parallelism: a.parallelism,
				DAG: &v1alpha12.DAGTemplate{
					Tasks: []v1alpha12.DAGTask{
						{
							Name:     pkg.ConvertToDNS1123(app.Name),
							Template: helmReleaseExecutor,
							Arguments: v1alpha12.Arguments{
								Parameters: []v1alpha12.Parameter{
									{
										Name:  helmReleaseArg,
										Value: strToStrPtr(base64.StdEncoding.EncodeToString([]byte(hrToYAML(hr)))),
									},
									{
										Name:  timeoutArg,
										Value: getTimeout(app.Spec.Release.Timeout),
									},
								},
							},
						},
					},
				},
			}

			ts = append(ts, tApp)
		}
	}
	return ts, nil
}

func (a *argo) generateSubchartAndAppDAGTasks(ctx context.Context, g *v1alpha1.ApplicationGroup, app *v1alpha1.Application, status *v1alpha1.ChartStatus, subchartsStatus map[string]v1alpha1.ChartStatus, repo, targetNS string) ([]v1alpha12.DAGTask, error) {
	if repo == "" {
		return nil, fmt.Errorf("repo arg must be a valid non-empty string")
	}

	// XXX (nitishm) Should this be set to nil if no subcharts are found??
	tasks := make([]v1alpha12.DAGTask, 0, len(app.Spec.Subcharts)+1)

	for _, sc := range app.Spec.Subcharts {
		scName := sc.Name

		isStaged := subchartsStatus[scName].Staged
		version := subchartsStatus[scName].Version

		hr, err := generateSubchartHelmRelease(*app, app.Name, scName, version, repo, targetNS, isStaged)
		if err != nil {
			return nil, err
		}
		hr.Annotations = map[string]string{
			"orkestra/parent-chart": app.Name,
		}
		hr.Labels = map[string]string{
			ChartLabelKey:  app.Name,
			OwnershipLabel: g.Name,
			HeritageLabel:  Project,
		}

		task := v1alpha12.DAGTask{
			Name:     pkg.ConvertToDNS1123(scName),
			Template: helmReleaseExecutor,
			Arguments: v1alpha12.Arguments{
				Parameters: []v1alpha12.Parameter{
					{
						Name:  helmReleaseArg,
						Value: strToStrPtr(base64.StdEncoding.EncodeToString([]byte(hrToYAML(*hr)))),
					},
					{
						Name:  timeoutArg,
						Value: getTimeout(app.Spec.Release.Timeout),
					},
				},
			},
			Dependencies: convertSliceToDNS1123(sc.Dependencies),
		}

		tasks = append(tasks, task)
	}

	hr := fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       "HelmRelease",
			APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      pkg.ConvertToDNS1123(app.Name),
			Namespace: app.Spec.Release.TargetNamespace,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   app.Spec.Chart.Name,
					Version: app.Spec.Chart.Version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      "HelmRepository",
						Name:      "chartmuseum",
						Namespace: workflowNamespace(),
					},
				},
			},
			Interval:        app.Spec.Release.Interval,
			ReleaseName:     pkg.ConvertToDNS1123(app.Name),
			TargetNamespace: app.Spec.Release.TargetNamespace,
			Timeout:         app.Spec.Release.Timeout,
			Values:          app.Spec.Release.Values,
		},
	}
	if app.Spec.Release.Install != nil {
		hr.Spec.Install = &fluxhelmv2beta1.Install{
			DisableWait: app.Spec.Release.DisableWaitForInstall(),
		}
	}
	if app.Spec.Release.Upgrade != nil {
		hr.Spec.Upgrade = &fluxhelmv2beta1.Upgrade{
			DisableWait: app.Spec.Release.DisableWaitForUpgrade(),
			Force:       app.Spec.Release.ForceForUpgrade(),
		}
	}
	if app.Spec.Release.Rollback != nil {
		hr.Spec.Rollback = &fluxhelmv2beta1.Rollback{
			DisableWait: app.Spec.Release.DisableWaitForRollback(),
		}
	}
	hr.Labels = map[string]string{
		ChartLabelKey:  app.Name,
		OwnershipLabel: g.Name,
		HeritageLabel:  Project,
	}

	// Force disable all subchart for the staged application chart
	// to prevent duplication and possible collision of deployed resources
	// Since the subchart should have been deployed in a prior DAG step,
	// we must not redeploy it along with the parent application chart.

	values := app.GetValues()
	for _, d := range app.Spec.Subcharts {
		values[d.Name] = map[string]interface{}{
			"enabled": false,
		}
	}
	if err := app.SetValues(values); err != nil {
		return nil, err
	}

	task := v1alpha12.DAGTask{
		Name:     pkg.ConvertToDNS1123(app.Name),
		Template: helmReleaseExecutor,
		Arguments: v1alpha12.Arguments{
			Parameters: []v1alpha12.Parameter{
				{
					Name:  helmReleaseArg,
					Value: strToStrPtr(base64.StdEncoding.EncodeToString([]byte(hrToYAML(hr)))),
				},
				{
					Name:  timeoutArg,
					Value: getTimeout(app.Spec.Release.Timeout),
				},
			},
		},
		Dependencies: func() (out []string) {
			for _, t := range tasks {
				out = append(out, pkg.ConvertToDNS1123(t.Name))
			}
			return out
		}(),
	}
	tasks = append(tasks, task)

	return tasks, nil
}

func updateWorkflowTemplates(wf *v1alpha12.Workflow, tpls ...v1alpha12.Template) {
	wf.Spec.Templates = append(wf.Spec.Templates, tpls...)
}

func defaultExecutor() v1alpha12.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--timeout", "{{inputs.parameters.timeout}}"}
	return v1alpha12.Template{
		Name:               helmReleaseExecutor,
		ServiceAccountName: workflowServiceAccountName(),
		Inputs: v1alpha12.Inputs{
			Parameters: []v1alpha12.Parameter{
				{
					Name: helmReleaseArg,
				},
				{
					Name:    timeoutArg,
					Default: &defaultTimeout,
				},
			},
		},
		Executor: &v1alpha12.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha12.Outputs{},
		Container: &corev1.Container{
			Name:  "executor",
			Image: "azureorkestra/executor:v0.1.0",
			Args:  executorArgs,
		},
	}
}

func defaultReverseExecutor() v1alpha12.Template {
	return v1alpha12.Template{
		Name:               helmReleaseReverseExecutor,
		ServiceAccountName: workflowServiceAccountName(),
		Inputs: v1alpha12.Inputs{
			Parameters: []v1alpha12.Parameter{
				{
					Name: "helmrelease",
				},
			},
		},
		Executor: &v1alpha12.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Resource: &v1alpha12.ResourceTemplate{
			// SetOwnerReference: true,
			Action:   "delete",
			Manifest: "{{inputs.parameters.helmrelease}}",
		},
	}
}

func hrToYAML(hr fluxhelmv2beta1.HelmRelease) string {
	b, err := yaml.Marshal(hr)
	if err != nil {
		return ""
	}

	return string(b)
}

func generateSubchartHelmRelease(a v1alpha1.Application, appName, scName, version, repo, targetNS string, isStaged bool) (*fluxhelmv2beta1.HelmRelease, error) {
	hr := &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       "HelmRelease",
			APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      pkg.ConvertToDNS1123(pkg.ToInitials(appName) + "-" + scName),
			Namespace: targetNS,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   pkg.ConvertToDNS1123(pkg.ToInitials(appName) + "-" + scName),
					Version: version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      "HelmRepository",
						Name:      "chartmuseum",
						Namespace: workflowNamespace(),
					},
				},
			},
			ReleaseName:     pkg.ConvertToDNS1123(scName),
			TargetNamespace: targetNS,
			Timeout:         a.Spec.Release.Timeout,
			Install: &fluxhelmv2beta1.Install{
				DisableWait: a.Spec.Release.DisableWaitForInstall(),
			},
			Upgrade: &fluxhelmv2beta1.Upgrade{
				DisableWait: a.Spec.Release.DisableWaitForUpgrade(),
			},
		},
	}

	val, err := subchartValues(scName, a.GetValues())
	if err != nil {
		return nil, err
	}
	hr.Spec.Values = val
	return hr, nil
}

func subchartValues(sc string, values map[string]interface{}) (*apiextensionsv1.JSON, error) {
	data := make(map[string]interface{})

	if scVals, ok := values[sc]; ok {
		if vv, ok := scVals.(map[string]interface{}); ok {
			for k, val := range vv {
				data[k] = val
			}
		}
	}

	if gVals, ok := values[valuesKeyGlobal]; ok {
		if vv, ok := gVals.(map[string]interface{}); ok {
			data[valuesKeyGlobal] = vv
		}
	}

	return v1alpha1.GetJSON(data)
}

func convertSliceToDNS1123(in []string) []string {
	out := []string{}
	for _, s := range in {
		out = append(out, pkg.ConvertToDNS1123(s))
	}
	return out
}

func strToStrPtr(in string) *string {
	return &in
}

func workflowNamespace() string {
	if ns, ok := os.LookupEnv("WORKFLOW_NAMESPACE"); ok {
		return ns
	}
	return "orkestra"
}

func workflowServiceAccountName() string {
	if sa, ok := os.LookupEnv("WORKFLOW_SERVICEACCOUNT_NAME"); ok {
		return sa
	}
	return "orkestra"
}

func getTimeout(t *v1.Duration) *string {
	if t == nil {
		return &defaultTimeout
	}
	tm := t.Duration.String()
	return &tm
}
