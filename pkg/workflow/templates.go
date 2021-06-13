package workflow

import (
	"encoding/base64"
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateTemplates(instance *v1alpha1.ApplicationGroup, options ClientOptions) (*v1alpha12.Template, []v1alpha12.Template, error) {
	if instance == nil {
		return nil, nil, fmt.Errorf("applicationGroup cannot be nil")
	}
	templates, err := generateAppDAGTemplates(instance, options.namespace, options.parallelism)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate application DAG templates : %w", err)
	}

	// Create the entry template from the app dag templates
	entryTemplate := &v1alpha12.Template{
		Name: EntrypointTemplateName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, len(instance.Spec.Applications)),
		},
		Parallelism: options.parallelism,
	}
	for i, tpl := range templates {
		entryTemplate.DAG.Tasks[i] = v1alpha12.DAGTask{
			Name:         utils.ConvertToDNS1123(tpl.Name),
			Template:     utils.ConvertToDNS1123(tpl.Name),
			Dependencies: utils.ConvertSliceToDNS1123(instance.Spec.Applications[i].Dependencies),
		}
	}
	return entryTemplate, templates, nil
}

func generateAppDAGTemplates(appGroup *v1alpha1.ApplicationGroup, namespace string, parallelism *int64) ([]v1alpha12.Template, error) {
	ts := make([]v1alpha12.Template, 0)

	for i, app := range appGroup.Spec.Applications {
		var hasSubcharts bool
		scStatus := appGroup.Status.Applications[i].Subcharts

		// Create Subchart DAG only when the application chart has dependencies
		if len(app.Spec.Subcharts) > 0 {
			hasSubcharts = true
			t := v1alpha12.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: parallelism,
			}

			t.DAG = &v1alpha12.DAGTemplate{}
			tasks, err := generateSubchartAndAppDAGTasks(appGroup, namespace, &app, scStatus)
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
								Namespace: namespace,
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
				OwnershipLabel: appGroup.Name,
				HeritageLabel:  Project,
			}

			tApp := v1alpha12.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: parallelism,
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

func generateSubchartAndAppDAGTasks(appGroup *v1alpha1.ApplicationGroup, namespace string, app *v1alpha1.Application, subchartsStatus map[string]v1alpha1.ChartStatus) ([]v1alpha12.DAGTask, error) {
	// XXX (nitishm) Should this be set to nil if no subcharts are found??
	tasks := make([]v1alpha12.DAGTask, 0, len(app.Spec.Subcharts)+1)

	for _, sc := range app.Spec.Subcharts {
		subchartName := sc.Name
		subchartVersion := subchartsStatus[subchartName].Version

		hr, err := generateSubchartHelmRelease(namespace, *app, subchartName, subchartVersion)
		if err != nil {
			return nil, err
		}
		hr.Annotations = map[string]string{
			v1alpha1.ParentChartAnnotation: app.Name,
		}
		hr.Labels = map[string]string{
			ChartLabelKey:  app.Name,
			OwnershipLabel: appGroup.Name,
			HeritageLabel:  Project,
		}

		task := v1alpha12.DAGTask{
			Name:     utils.ConvertToDNS1123(subchartName),
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
						Namespace: namespace,
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
		OwnershipLabel: appGroup.Name,
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

func generateSubchartHelmRelease(namespace string, a v1alpha1.Application, subchartName, version string) (*fluxhelmv2beta1.HelmRelease, error) {
	chName := utils.GetSubchartName(a.Spec.Chart.Name, subchartName)
	hr := &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      chName,
			Namespace: a.Spec.Release.TargetNamespace,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   chName,
					Version: version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: namespace,
					},
				},
			},
			ReleaseName:     utils.ConvertToDNS1123(subchartName),
			TargetNamespace: a.Spec.Release.TargetNamespace,
			Timeout:         a.Spec.Release.Timeout,
			Install:         a.Spec.Release.Install,
			Upgrade:         a.Spec.Release.Upgrade,
			Rollback:        a.Spec.Release.Rollback,
			Uninstall:       a.Spec.Release.Uninstall,
		},
	}

	val, err := subchartValues(subchartName, a.GetValues())
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

func getTaskNamesFromHelmReleases(bucket []fluxhelmv2beta1.HelmRelease) []string {
	out := []string{}
	for _, hr := range bucket {
		out = append(out, utils.ConvertToDNS1123(hr.GetReleaseName()+"-"+(hr.Namespace)))
	}
	return out
}
