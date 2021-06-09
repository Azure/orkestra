package helpers

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/utils"
	"github.com/Azure/Orkestra/pkg/workflow"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	"helm.sh/helm/v3/pkg/chart"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ErrInvalidSpec = fmt.Errorf("custom resource spec is invalid")
	// ErrRequeue describes error while requeuing
	ErrRequeue                    = fmt.Errorf("(transitory error) Requeue-ing resource to try again")
	ErrWorkflowInFailureStatus    = errors.New("workflow in failure status")
	ErrHelmReleaseInFailureStatus = errors.New("helmrelease in failure status")

	dummyConfigmapYAMLSpec = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-dummy
  namespace: {{ .Release.Namespace }} 
data:
  name: {{ .Chart.Name }}
  version : {{ .Chart.Version }}
`
	dummyConfigmapYAMLName = "templates/dummy-configmap.yaml"
)

type ReconcileHelper struct {
	client.Client
	logr.Logger
	Instance       *v1alpha1.ApplicationGroup
	Engine         workflow.Engine
	RegistryClient *registry.Client

	RegistryOptions RegistryClientOptions
}

type RegistryClientOptions struct {
	StagingRepoName         string
	TargetDir               string
	CleanupDownloadedCharts bool
	stagingDirecotry        string
}

func (helper *ReconcileHelper) CreateOrUpdate(ctx context.Context) error {
	helper.Logger = helper.Logger.WithValues(v1alpha1.AppGroupNameKey, helper.Instance.Name)
	helper.V(3).Info("Reconciling ApplicationGroup object")

	if len(helper.Instance.Spec.Applications) == 0 {
		helper.Error(ErrInvalidSpec, "ApplicationGroup must list atleast one Application")
		err := fmt.Errorf("application group must list atleast one Application : %w", ErrInvalidSpec)
		return err
	}

	if err := helper.reconcileApplications(); err != nil {
		helper.Error(err, "failed to reconcile the applications")
		err = fmt.Errorf("failed to reconcile the applications : %w", err)
		return err
	}
	// Generate the Workflow object to submit to Argo
	if err := helper.generateWorkflow(ctx, helper.Logger, helper.Instance); err != nil {
		helper.Error(err, "failed to reconcile ApplicationGroup instance")
		return err
	}
	helper.Instance.Status.ObservedGeneration = helper.Instance.Generation
	return nil
}

func (helper *ReconcileHelper) Rollback(ctx context.Context, patch client.Patch, err error) (ctrl.Result, error) {
	// If this is a HelmRelease failure then we must remediate by cleaning up
	// all the helm releases deployed by the workflow and helm operator
	if errors.Is(err, ErrHelmReleaseInFailureStatus) {
		// Delete the HelmRelease(s) - parent and subchart(s)
		// Lookup charts using the label selector.
		// Example: chart=kafka-dev,heritage=orkestra,owner=dev, where chart=<top-level-chart>
		helper.Info("Remediating the applicationgroup with helmrelease failure status")
		for _, app := range helper.Instance.Status.Applications {
			helmReleaseReason := meta.GetResourceCondition(&app.ChartStatus, meta.ReadyCondition).Reason
			if meta.IsFailedHelmReason(helmReleaseReason) {
				listOption := client.MatchingLabels{
					workflow.OwnershipLabel: helper.Instance.Name,
					workflow.HeritageLabel:  workflow.Project,
					workflow.ChartLabelKey:  app.Name,
				}
				helmReleases := fluxhelmv2beta1.HelmReleaseList{}
				err = helper.List(ctx, &helmReleases, listOption)
				if err != nil {
					helper.Error(err, "failed to find generated HelmRelease instances")
					return reconcile.Result{}, nil
				}

				err = helper.rollbackFailedHelmReleases(ctx, helmReleases.Items)
				if err != nil {
					helper.Error(err, "failed to rollback failed HelmRelease instances")
					return reconcile.Result{}, nil
				}
			}
		}
	}
	// mark the object as requiring rollback so that we can rollback
	// to the previous versions of all the applications in the ApplicationGroup
	// using the last successful spec
	helper.Info("Rolling back to last successful application group spec")
	helper.Instance.Spec = *helper.Instance.GetLastSuccessful()
	if err := helper.Patch(ctx, helper.Instance, patch); err != nil {
		helper.Error(err, "failed to update ApplicationGroup instance while rolling back")
		return ctrl.Result{}, err
	}
	return reconcile.Result{RequeueAfter: v1alpha1.DefaultProgressingRequeue}, nil
}

func (helper *ReconcileHelper) Reverse(ctx context.Context) (ctrl.Result, error) {
	if workflow := helper.GetWorkflow(ctx); workflow != nil {
		helper.Info("cleaning up the workflow object")
		if err := helper.cleanupWorkflow(ctx, helper.Logger, workflow); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, nil
}

func (helper *ReconcileHelper) GetWorkflow(ctx context.Context) *v1alpha12.Workflow {
	wfs := v1alpha12.WorkflowList{}
	listOption := client.MatchingLabels{
		workflow.OwnershipLabel: helper.Instance.Name,
		workflow.HeritageLabel:  workflow.Project,
	}
	_ = helper.List(ctx, &wfs, listOption)

	if len(wfs.Items) != 0 {
		return &wfs.Items[0]
	}
	return nil
}

func (helper *ReconcileHelper) reconcileApplications() error {
	// Init the application status every time we re-reconcile the applications
	initAppStatus(helper.Instance)

	// Pull and conditionally stage application & dependency charts
	for i, application := range helper.Instance.Spec.Applications {
		ll := helper.WithValues("application", application.Name)
		ll.V(3).Info("performing chart actions")

		repoCfg, err := registry.GetHelmRepoConfig(&application, helper.Client)
		if err != nil {
			err = fmt.Errorf("failed to get repo configuration for repo at URL %s: %w", application.Spec.Chart.URL, err)
			ll.Error(err, "failed to add helm repo ")
			return err
		}

		err = helper.RegistryClient.AddRepo(repoCfg)
		if err != nil {
			err = fmt.Errorf("failed to add helm repo at URL %s: %w", application.Spec.Chart.URL, err)
			ll.Error(err, "failed to add helm repo ")
			return err
		}

		name := application.Spec.Chart.Name
		version := application.Spec.Chart.Version
		repoKey := application.Name

		fpath, appCh, err := helper.RegistryClient.PullChart(ll, repoKey, name, version)
		defer func() {
			if helper.RegistryOptions.CleanupDownloadedCharts {
				os.Remove(fpath)
			}
		}()
		if err != nil || appCh == nil {
			err = fmt.Errorf("failed to pull application chart %s/%s:%s : %w", repoKey, name, version, err)
			ll.Error(err, "failed to pull application chart")
			return err
		}

		if appCh.Dependencies() != nil {
			// take account of all embedded subcharts found in the application chart
			embeddedSubcharts := make(map[string]bool)
			for _, d := range appCh.Dependencies() {
				embeddedSubcharts[d.Name()] = true
			}

			// Remove all explicit subchart entries from tracking map
			for _, d := range helper.Instance.Spec.Applications[i].Spec.Subcharts {
				delete(embeddedSubcharts, d.Name)
			}

			// Use the remaining set of dependencies that are not explicitly declared and
			// add them to the groups application spec's subcharts list
			for name := range embeddedSubcharts {
				helper.Instance.Spec.Applications[i].Spec.Subcharts = append(helper.Instance.Spec.Applications[i].Spec.Subcharts, v1alpha1.DAG{Name: name})
			}

			stagingRepoName := helper.RegistryOptions.StagingRepoName
			// Package and push all application subcharts staging registry
			for _, sc := range appCh.Dependencies() {
				chartStatus := v1alpha1.ChartStatus{
					Version: sc.Metadata.Version,
					Staged:  false,
					Error:   "",
				}

				// copy the subchart
				scc := &chart.Chart{}
				_ = copier.Copy(scc, sc)

				if err := scc.Validate(); err != nil {
					ll.Error(err, "failed to validate application subchart for staging registry")
					err = fmt.Errorf("failed to validate application subchart for staging registry : %w", err)
					chartStatus.Error = err.Error()
					helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				// Copy over all non yaml files from parent chart templates to subchart templates
				for _, f := range appCh.Templates {
					if !utils.IsFileYaml(f.Name) {
						t := &chart.File{}
						_ = copier.Copy(t, f)
						t.Name = utils.AddAppChartNameToFile(t.Name, appCh.Name())
						scc.Templates = append(scc.Templates, t)
					}
				}

				scc.Metadata.Name = utils.ConvertToDNS1123(utils.ToInitials(appCh.Metadata.Name) + "-" + scc.Metadata.Name)
				path, err := registry.SaveChartPackage(scc, helper.getStagingDirectory())
				if err != nil {
					ll.Error(err, "failed to save subchart package as tgz")
					err = fmt.Errorf("failed to save subchart package as tgz at location %s : %w", path, err)
					chartStatus.Error = err.Error()
					helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				err = helper.RegistryClient.PushChart(ll, stagingRepoName, path, scc)
				if err != nil {
					ll.Error(err, "failed to push application subchart to staging registry")
					err = fmt.Errorf("failed to push application subchart to staging registry : %w", err)
					chartStatus.Error = err.Error()
					helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				chartStatus.Staged = true

				helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
			}
		}

		// Unset dependencies by disabling them.
		// Using appCh.SetDependencies() does not cut it since some charts rely on subcharts for tpl helpers
		// provided in the charts directory.
		// IMPORTANT: This expects charts to follow best practices to allow enabling and disabling subcharts
		// See: https://helm.sh/docs/topics/charts/ #Chart Dependencies
		for _, dep := range appCh.Metadata.Dependencies {
			// Disable subchart through metadata
			dep.Enabled = false
			// Precautionary - overwrite values with subcharts disabled
			appCh.Values[dep.Name] = map[string]interface{}{
				"enabled": false,
			}
		}

		templateHasYAML, err := utils.TemplateContainsYaml(appCh)
		if err != nil {
			ll.Error(err, "chart templates directory yaml check failed")
			err = fmt.Errorf("chart templates directory yaml check failed : %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		// If the parent chart doesnt contain any templates and all subcharts (if any) have been disabled we must create a dummy yaml to circumvent https://github.com/helm/helm/issues/4670
		if appCh.Templates == nil || len(appCh.Templates) == 0 || !templateHasYAML {
			if appCh.Templates == nil {
				appCh.Templates = make([]*chart.File, 0)
			}
			dummy := &chart.File{
				Name: dummyConfigmapYAMLName,
				Data: []byte(dummyConfigmapYAMLSpec),
			}
			appCh.Templates = append(appCh.Templates, dummy)
		}

		if err := appCh.Validate(); err != nil {
			ll.Error(err, "failed to validate application chart for staging registry")
			err = fmt.Errorf("failed to validate application chart for staging registry : %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		appCh.Metadata.Name = utils.ConvertToDNS1123(appCh.Metadata.Name)

		_, err = registry.SaveChartPackage(appCh, helper.getStagingDirectory())
		if err != nil {
			ll.Error(err, "failed to save modified app chart to filesystem")
			err = fmt.Errorf("failed to save modified app chart to filesystem : %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		// Replace existing chart with modified chart
		chartPath := helper.getChartPath(application.Spec.Chart.Name, appCh.Metadata.Version)
		err = helper.RegistryClient.PushChart(ll, helper.RegistryOptions.StagingRepoName, chartPath, appCh)
		defer func() {
			if helper.RegistryOptions.CleanupDownloadedCharts {
				os.Remove(chartPath)
			}
		}()
		if err != nil {
			ll.Error(err, "failed to push modified application chart to staging registry")
			err = fmt.Errorf("failed to push modified application chart to staging registry : %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		helper.Instance.Status.Applications[i].ChartStatus.Staged = true
	}
	return nil
}

func (helper *ReconcileHelper) rollbackFailedHelmReleases(ctx context.Context, hrs []fluxhelmv2beta1.HelmRelease) error {
	for _, hr := range hrs {
		err := utils.HelmRollback(hr.Spec.ReleaseName, hr.Spec.TargetNamespace)
		if err != nil {
			return err
		}
		err = helper.Delete(ctx, &hr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (helper *ReconcileHelper) getStagingDirectory() string {
	return helper.RegistryOptions.TargetDir + "/" + helper.RegistryOptions.StagingRepoName
}

func (helper *ReconcileHelper) getChartPath(name, version string) string {
	return helper.getStagingDirectory() + "/" + utils.ConvertToDNS1123(name) + "-" + version + ".tgz"
}
