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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The set of executor actions which can be performed on a helmrelease object
	Install ExecutorAction = "install"
	Delete  ExecutorAction = "delete"
)

// ExecutorAction defines the set of executor actions which can be performed on a helmrelease object
type ExecutorAction string

func getTaskNamesFromHelmReleases(bucket []fluxhelmv2beta1.HelmRelease) []string {
	out := []string{}
	for _, hr := range bucket {
		out = append(out, utils.ConvertToDNS1123(hr.GetReleaseName()+"-"+(hr.Namespace)))
	}
	return out
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
			Name:         utils.ConvertToDNS1123(tpl.Name),
			Template:     utils.ConvertToDNS1123(tpl.Name),
			Dependencies: utils.ConvertSliceToDNS1123(g.Spec.Applications[i].Dependencies),
		}
	}

	return nil
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
			v1alpha1.ParentChartAnnotation: app.Name,
		}
		hr.Labels = map[string]string{
			ChartLabelKey:  app.Name,
			OwnershipLabel: g.Name,
			HeritageLabel:  Project,
		}

		task := v1alpha12.DAGTask{
			Name:     utils.ConvertToDNS1123(scName),
			Template: HelmReleaseExecutorName,
			Arguments: v1alpha12.Arguments{
				Parameters: []v1alpha12.Parameter{
					{
						Name:  HelmReleaseArg,
						Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(*hr)))),
					},
					{
						Name:  TimeoutArg,
						Value: getTimeout(app.Spec.Release.Timeout),
					},
				},
			},
			Dependencies: utils.ConvertSliceToDNS1123(sc.Dependencies),
		}

		tasks = append(tasks, task)
	}

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
		Dependencies: func() (out []string) {
			for _, t := range tasks {
				out = append(out, utils.ConvertToDNS1123(t.Name))
			}
			return out
		}(),
	}
	tasks = append(tasks, task)

	return tasks, nil
}

func defaultExecutor(tplName string, action ExecutorAction) v1alpha12.Template {
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--timeout", "{{inputs.parameters.timeout}}", "--interval", "10s"}
	return v1alpha12.Template{
		Name:               tplName,
		ServiceAccountName: workflowServiceAccountName(),
		Inputs: v1alpha12.Inputs{
			Parameters: []v1alpha12.Parameter{
				{
					Name: HelmReleaseArg,
				},
				{
					Name:    TimeoutArg,
					Default: utils.ToStrPtr(DefaultTimeout),
				},
			},
		},
		Executor: &v1alpha12.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha12.Outputs{},
		Container: &corev1.Container{
			Name:  ExecutorName,
			Image: fmt.Sprintf("%s:%s", ExecutorImage, ExecutorImageTag),
			Args:  executorArgs,
		},
	}
}

func generateSubchartHelmRelease(a v1alpha1.Application, appName, scName, version, repo, targetNS string, isStaged bool) (*fluxhelmv2beta1.HelmRelease, error) {
	hr := &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.ConvertToDNS1123(utils.ToInitials(appName) + "-" + scName),
			Namespace: targetNS,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   utils.ConvertToDNS1123(utils.ToInitials(appName) + "-" + scName),
					Version: version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: workflowNamespace(),
					},
				},
			},
			ReleaseName:     utils.ConvertToDNS1123(scName),
			TargetNamespace: targetNS,
			Timeout:         a.Spec.Release.Timeout,
			Install:         a.Spec.Release.Install,
			Upgrade:         a.Spec.Release.Upgrade,
			Rollback:        a.Spec.Release.Rollback,
			Uninstall:       a.Spec.Release.Uninstall,
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

	if gVals, ok := values[ValuesKeyGlobal]; ok {
		if vv, ok := gVals.(map[string]interface{}); ok {
			data[ValuesKeyGlobal] = vv
		}
	}

	return v1alpha1.GetJSON(data)
}
