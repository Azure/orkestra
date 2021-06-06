package workflow

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (a ForwardEngine) Generate(ctx context.Context, l logr.Logger, g *v1alpha1.ApplicationGroup) error {
	if g == nil {
		l.Error(nil, "ApplicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	a.workflow = initWorkflowObject()

	// Set name and namespace based on the input application group
	a.workflow.Name = g.Name
	a.workflow.Namespace = workflowNamespace()

	err := a.generateWorkflow(ctx, g)
	if err != nil {
		l.Error(err, "failed to generate workflow")
		return fmt.Errorf("failed to generate argo workflow : %w", err)
	}

	updateWorkflowTemplates(a.wf, adt...)

	err = updateAppGroupDAG(g, &entry, adt)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}
	updateWorkflowTemplates(a.wf, entry)

	// TODO: Add the executor template
	// This should eventually be configurable
	updateWorkflowTemplates(a.wf, defaultExecutor(HelmReleaseExecutorName, Install))

	return nil
}

func (a ForwardEngine) Submit(ctx context.Context, l logr.Logger, g *v1alpha1.ApplicationGroup) error {
	if a.workflow == nil {
		l.Error(nil, "workflow object cannot be nil")
		return fmt.Errorf("workflow object cannot be nil")
	}

	if g == nil {
		l.Error(nil, "applicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	namespaces := []string{}
	// Add namespaces we need to create while removing duplicates
	for _, app := range g.Spec.Applications {
		found := false
		for _, namespace := range namespaces {
			if app.Spec.Release.TargetNamespace == namespace {
				found = true
				break
			}
		}
		if !found {
			namespaces = append(namespaces, app.Spec.Release.TargetNamespace)
		}
	}

	// Create any of the target namespaces
	for _, namespace := range namespaces {
		ns := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: namespace,
			},
		}
		if err := controllerutil.SetControllerReference(g, ns, a.Scheme()); err != nil {
			return fmt.Errorf("failed to set OwnerReference for Namespace %s : %w", ns.Name, err)
		}
		if err := a.Create(ctx, ns); !errors.IsAlreadyExists(err) && err != nil {
			return fmt.Errorf("failed to CREATE namespace %s object : %w", ns.Name, err)
		}
	}

	// Create the Workflow
	a.workflow.Labels[OwnershipLabel] = g.Name
	if err := controllerutil.SetControllerReference(g, a.workflow, a.Scheme()); err != nil {
		l.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := a.Create(ctx, a.workflow); !errors.IsAlreadyExists(err) && err != nil {
		l.Error(err, "failed to CREATE argo workflow object")
		return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
	} else if errors.IsAlreadyExists(err) {
		// If the workflow needs an update, delete the previous workflow and apply the new one
		// Argo Workflow does not rerun the workflow on UPDATE, so intead we cleanup and reapply
		if err := a.Delete(ctx, a.workflow); err != nil {
			l.Error(err, "failed to DELETE argo workflow object")
			return fmt.Errorf("failed to DELETE argo workflow object : %w", err)
		}
		if err := controllerutil.SetControllerReference(g, a.workflow, a.Scheme()); err != nil {
			l.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
			return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
		}
		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		if err := a.Create(ctx, a.workflow); err != nil {
			l.Error(err, "failed to CREATE argo workflow object")
			return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
		}
	}
	return nil
}

func generateWorkflow(ctx context.Context, g *v1alpha1.ApplicationGroup) error {
	// Generate the Entrypoint template and Application Group DAG
	err := generateAppGroupTpls(ctx, g)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}

	return nil
}

func generateAppGroupTpls(ctx context.Context, g *v1alpha1.ApplicationGroup, parallelism *int64) error {
	if a.wf == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	if g == nil {
		return fmt.Errorf("applicationGroup cannot be nil")
	}

	entry := v1alpha12.Template{
		Name: EntrypointTemplateName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, len(g.Spec.Applications)),
			// TBD (nitishm): Do we need to failfast?
			// FailFast: true
		},
		Parallelism: parallelism,
	}

	adt, err := generateAppDAGTemplates(ctx, g, a.stagingRepoURL)
	if err != nil {
		return fmt.Errorf("failed to generate application DAG templates : %w", err)
	}
	return nil
}

func generateAppDAGTemplates(ctx context.Context, g *v1alpha1.ApplicationGroup, repo string) ([]v1alpha12.Template, error) {
	ts := make([]v1alpha12.Template, 0)

	for i, app := range g.Spec.Applications {
		var hasSubcharts bool
		appStatus := &g.Status.Applications[i].ChartStatus
		scStatus := g.Status.Applications[i].Subcharts

		// Create Subchart DAG only when the application chart has dependencies
		if len(app.Spec.Subcharts) > 0 {
			hasSubcharts = true
			t := v1alpha12.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
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
					Kind:       fluxhelmv2beta1.HelmReleaseKind,
					APIVersion: fluxhelmv2beta1.GroupVersion.String(),
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      utils.ConvertToDNS1123(app.Name),
					Namespace: app.Spec.Release.TargetNamespace,
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   utils.ConvertToDNS1123(app.Spec.Chart.Name),
							Version: app.Spec.Chart.Version,
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      fluxsourcev1beta1.HelmRepositoryKind,
								Name:      ChartMuseumName,
								Namespace: workflowNamespace(),
							},
						},
					},
					Interval:        app.Spec.Release.Interval,
					ReleaseName:     utils.ConvertToDNS1123(app.Name),
					TargetNamespace: app.Spec.Release.TargetNamespace,
					Timeout:         app.Spec.Release.Timeout,
					Values:          app.Spec.Release.Values,
					Install:         app.Spec.Release.Install,
					Upgrade:         app.Spec.Release.Upgrade,
					Rollback:        app.Spec.Release.Rollback,
					Uninstall:       app.Spec.Release.Uninstall,
				},
			}
			hr.Labels = map[string]string{
				ChartLabelKey:  app.Name,
				OwnershipLabel: g.Name,
				HeritageLabel:  Project,
			}

			tApp := v1alpha12.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: a.parallelism,
				DAG: &v1alpha12.DAGTemplate{
					Tasks: []v1alpha12.DAGTask{
						{
							Name:     utils.ConvertToDNS1123(app.Name),
							Template: HelmReleaseExecutorName,
							Arguments: v1alpha12.Arguments{
								Parameters: []v1alpha12.Parameter{
									{
										Name:  HelmReleaseArg,
										Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(hr)))),
									},
									{
										Name:  TimeoutArg,
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