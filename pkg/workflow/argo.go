package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	helmopv1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	project           = "orkestra"
	ownershipLabel    = "owner"
	heritageLabel     = "heritage"
	argoAPIVersion    = "argoproj.io/v1alpha1"
	argoKind          = "Workflow"
	entrypointTplName = "entry"

	helmReleaseArg = "helmrelease"

	valuesKeyGlobal = "global"
)

type argo struct {
	scheme *runtime.Scheme
	cli    client.Client
	wf     *v1alpha12.Workflow

	stagingRepoURL string
}

// Argo is blah blah
func Argo(scheme *runtime.Scheme, c client.Client, r string) *argo {

	return &argo{
		cli:            c,
		stagingRepoURL: r,
	}
}

func (a *argo) initWorkflowObject() {
	a.wf = &v1alpha12.Workflow{}

	a.wf.APIVersion = argoAPIVersion
	a.wf.Kind = argoKind

	// Entry point is the entry node into the Application Group DAG
	a.wf.Spec.Entrypoint = entrypointTplName

	// Initialize the Templates slice
	a.wf.Spec.Templates = make([]v1alpha12.Template, 0)
}

func (a *argo) Generate(ctx context.Context, l logr.Logger, ns string, g *v1alpha1.ApplicationGroup, apps []*v1alpha1.Application) error {
	if g == nil {
		l.Error(nil, "ApplicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	if apps == nil {
		l.Error(nil, "applications slice cannot be nil")
		return fmt.Errorf("applications slice cannot be nil")
	}

	if len(apps) != len(g.Spec.Applications) {
		l.Error(nil, "application len mismatch")
		return fmt.Errorf("len application objects [%v] do not match len entries applicationgroup.spec.applications %v", len(apps), len(g.Spec.Applications))
	}

	a.initWorkflowObject()

	// Set name and namespace based on the input application group
	a.wf.Name = g.Name
	a.wf.Namespace = ns

	err := a.generateWorkflow(g, apps)
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

	obj.Labels[ownershipLabel] = g.Name
	obj.Labels[heritageLabel] = project

	err := a.cli.Get(ctx, types.NamespacedName{Namespace: a.wf.Namespace, Name: a.wf.Name}, obj)
	if err != nil {
		if errors.IsNotFound(err) && !errors.IsAlreadyExists(err) {
			// Add OwnershipReference
			err = controllerutil.SetControllerReference(g, a.wf, a.scheme)
			if err != nil {
				l.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
				return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
			}

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
	return nil
}

func (a *argo) generateWorkflow(g *v1alpha1.ApplicationGroup, apps []*v1alpha1.Application) error {
	// Generate the Entrypoint template and Application Group DAG
	err := a.generateAppGroupTpls(g, apps)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}

	return nil
}

func (a *argo) generateAppGroupTpls(g *v1alpha1.ApplicationGroup, apps []*v1alpha1.Application) error {
	if a.wf == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	if g == nil {
		return fmt.Errorf("applicationGroup cannot be nil")
	}

	entry := v1alpha12.Template{
		Name: entrypointTplName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, len(apps)),
			// TBD (nitishm): Do we need to failfast?
			// FailFast: true
		},
	}

	err := updateAppGroupDAG(&entry, apps)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}
	a.updateWorkflowTemplates(entry)

	adt, err := generateAppDAGTemplates(apps, a.stagingRepoURL)
	if err != nil {
		return fmt.Errorf("failed to generate application DAG templates : %w", err)
	}
	a.updateWorkflowTemplates(adt...)

	// TODO: Add the executor template
	// This should eventually be configurable
	a.updateWorkflowTemplates(defaultExecutor())

	return nil
}

func updateAppGroupDAG(entry *v1alpha12.Template, apps []*v1alpha1.Application) error {
	if entry == nil {
		return fmt.Errorf("entry template cannot be nil")
	}

	for i, app := range apps {
		entry.DAG.Tasks[i] = v1alpha12.DAGTask{
			Name:     app.Name,
			Template: app.Name,
		}
	}

	return nil
}

func generateAppDAGTemplates(apps []*v1alpha1.Application, repo string) ([]v1alpha12.Template, error) {
	ts := make([]v1alpha12.Template, 0, len(apps))

	for _, app := range apps {
		// Create Subchart DAG only when the application chart has dependencies
		if len(app.Spec.Subcharts) > 0 {
			t := v1alpha12.Template{
				Name: app.Name,
			}

			t.DAG = &v1alpha12.DAGTemplate{}
			tasks, err := generateSubchartDAGTasks(app, repo)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Application Template DAG tasks : %w", err)
			}

			t.DAG.Tasks = tasks

			ts = append(ts, t)
		}

		hr := helmopv1.HelmRelease{
			TypeMeta: v1.TypeMeta{
				Kind:       "HelmRelease",
				APIVersion: "helm.fluxcd.io/v1",
			},
		}

		hr.Name = app.Name
		hr.Spec = app.Spec.HelmReleaseSpec

		// TODO (nitishm)
		// FIXME: Do not assume application chart will get pushed to the staging registry for every application
		// Application charts that do not specify subchart dependencies should generate the HelmRelease with the
		// RepoURL pointing to the primary chart
		// NOTE: For now let's assume that each chart is being pushed to staging
		hr.Spec.RepoURL = repo

		// tStaging is the Application chart HelmRelease that points to the staged Application (with dependencies set to null)
		tStaging := v1alpha12.Template{
			Name: app.Name,
			Arguments: v1alpha12.Arguments{
				Parameters: []v1alpha12.Parameter{
					{
						Name:  helmReleaseArg,
						Value: strToStrPtr(hrToYAML(hr)),
					},
				},
			},
		}

		ts = append(ts, tStaging)
	}

	return ts, nil
}

func generateSubchartDAGTasks(app *v1alpha1.Application, repo string) ([]v1alpha12.DAGTask, error) {
	if repo == "" {
		return nil, fmt.Errorf("repo arg must be a valid non-empty string")
	}

	// TODO (nitishm)
	// TBD: Should this be set to nil if no subcharts are found??
	tasks := make([]v1alpha12.DAGTask, 0, len(app.Spec.Subcharts))

	for _, sc := range app.Spec.Subcharts {
		hr := generateSubchartHelmRelease(app.Spec.HelmReleaseSpec, sc.Name, repo)
		task := v1alpha12.DAGTask{
			Name: sc.Name,
			Arguments: v1alpha12.Arguments{
				Parameters: []v1alpha12.Parameter{
					{
						Name:  helmReleaseArg,
						Value: strToStrPtr(hrToYAML(hr)),
					},
				},
			},
			Dependencies: sc.Dependencies,
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (a *argo) updateWorkflowTemplates(tpls ...v1alpha12.Template) {
	a.wf.Spec.Templates = append(a.wf.Spec.Templates, tpls...)
}

func defaultExecutor() v1alpha12.Template {
	return v1alpha12.Template{
		Name: "helmrelease-executor",
		Inputs: v1alpha12.Inputs{
			Parameters: []v1alpha12.Parameter{
				{
					Name: "helmrelease",
				},
			},
		},
		Outputs: v1alpha12.Outputs{},
		Resource: &v1alpha12.ResourceTemplate{
			Action:           "create",
			Manifest:         "{{inputs.parameters.helmrelease}}",
			SuccessCondition: "status.phase == Succeeded",
		},
	}
}

func strToStrPtr(s string) *string {
	return &s
}

func hrToYAML(hr helmopv1.HelmRelease) string {
	b, err := json.MarshalIndent(hr, "", "  ")
	if err != nil {
		return ""
	}

	return string(b)
}

func generateSubchartHelmRelease(a helmopv1.HelmReleaseSpec, sc, repo string) helmopv1.HelmRelease {
	hr := helmopv1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       "HelmRelease",
			APIVersion: "helm.fluxcd.io/v1",
		},
		Spec: helmopv1.HelmReleaseSpec{
			ChartSource: helmopv1.ChartSource{
				RepoChartSource: &helmopv1.RepoChartSource{},
			},
		},
	}

	hr.Name = sc

	hr.Spec.RepoURL = repo
	hr.Spec.Name = sc
	hr.Spec.Version = a.Version

	hr.Spec.Values = subchartValues(sc, a.Values)

	return hr
}

func subchartValues(sc string, av helmopv1.HelmValues) helmopv1.HelmValues {
	v := helmopv1.HelmValues{
		Data: make(map[string]interface{}),
	}

	if scVals, ok := av.Data[sc]; ok {
		if vv, ok := scVals.(map[string]interface{}); ok {
			for k, val := range vv {
				v.Data[k] = val
			}
		}
	}

	if gVals, ok := av.Data[valuesKeyGlobal]; ok {
		if vv, ok := gVals.(map[string]interface{}); ok {
			v.Data[valuesKeyGlobal] = vv
		}
	}

	return v
}
