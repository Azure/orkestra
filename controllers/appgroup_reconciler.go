package controllers

import (
	"context"
	"errors"
	"os"

	"github.com/Azure/Orkestra/pkg/workflow"

	"github.com/Azure/Orkestra/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	ErrInvalidSpec = fmt.Errorf("custom resource spec is invalid")
	// ErrRequeue describes error while requeuing
	ErrRequeue = fmt.Errorf("(transitory error) Requeue-ing resource to try again")

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

func (r *ApplicationGroupReconciler) reconcileCreateOrUpdate(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch) error {
	instance.Progressing()
	if err := r.Status().Patch(ctx, instance, patch); err != nil {
		return err
	}

	r.Log = r.Log.WithValues(v1alpha1.AppGroupNameKey, instance.Name)
	r.Log.V(3).Info("Reconciling ApplicationGroup object")

	if len(instance.Spec.Applications) == 0 {
		r.Log.Error(ErrInvalidSpec, "ApplicationGroup must list atleast one Application")
		err := fmt.Errorf("application group must list atleast one Application : %w", ErrInvalidSpec)
		return err
	}

	if err := r.reconcileApplications(r.Log, instance); err != nil {
		r.Log.Error(err, "failed to reconcile the applications")
		err = fmt.Errorf("failed to reconcile the applications : %w", err)
		return err
	}
	// Generate the Workflow object to submit to Argo
	engine, err := r.EngineBuilder.Forward(instance).Build()
	if err != nil {
		return err
	}
	if err := workflow.Run(ctx, engine); err != nil {
		r.Log.Error(err, "failed to reconcile ApplicationGroup instance")
		return err
	}
	instance.Status.ObservedGeneration = instance.Generation
	return nil
}

func (r *ApplicationGroupReconciler) reconcileRollback(ctx context.Context, instance *v1alpha1.ApplicationGroup, patch client.Patch) error {
	if instance.GetLastSuccessful() == nil {
		err := errors.New("failed to rollback ApplicationGroup instance due to missing last successful applicationgroup annotation")
		r.Log.Error(err, "")
		return err
	}
	r.Log.Info("Rolling back to last successful application group spec")
	instance.Spec = *instance.GetLastSuccessful()
	if err := r.Patch(ctx, instance, patch); err != nil {
		r.Log.Error(err, "failed to update ApplicationGroup instance while rolling back")
		return err
	}
	instance.Progressing()
	if err := r.Status().Patch(ctx, instance, patch); err != nil {
		return err
	}
	return nil
}

func (r *ApplicationGroupReconciler) reconcileApplications(l logr.Logger, appGroup *v1alpha1.ApplicationGroup) error {
	stagingDir := r.TargetDir + "/" + r.StagingRepoName

	// Init the application status every time we re-reconcile the applications
	InitAppStatus(appGroup)

	// Pull and conditionally stage application & dependency charts
	for i, application := range appGroup.Spec.Applications {
		ll := l.WithValues("application", application.Name)
		ll.V(3).Info("performing chart actions")

		repoCfg, err := registry.GetHelmRepoConfig(&application, r.Client)
		if err != nil {
			err = fmt.Errorf("failed to get repo configuration for repo at URL %s: %w", application.Spec.Chart.URL, err)
			ll.Error(err, "failed to add helm repo ")
			return err
		}

		err = r.RegistryClient.AddRepo(repoCfg)
		if err != nil {
			err = fmt.Errorf("failed to add helm repo at URL %s: %w", application.Spec.Chart.URL, err)
			ll.Error(err, "failed to add helm repo ")
			return err
		}

		name := application.Spec.Chart.Name
		version := application.Spec.Chart.Version
		repoKey := application.Name

		fpath, appCh, err := r.RegistryClient.PullChart(ll, repoKey, name, version)
		defer func() {
			if r.CleanupDownloadedCharts {
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
			for _, d := range appGroup.Spec.Applications[i].Spec.Subcharts {
				delete(embeddedSubcharts, d.Name)
			}

			// Use the remaining set of dependencies that are not explicitly declared and
			// add them to the groups application spec's subcharts list
			for name := range embeddedSubcharts {
				appGroup.Spec.Applications[i].Spec.Subcharts = append(appGroup.Spec.Applications[i].Spec.Subcharts, v1alpha1.DAG{Name: name})
			}

			stagingRepoName := r.StagingRepoName
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
					appGroup.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
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
				path, err := registry.SaveChartPackage(scc, stagingDir)
				if err != nil {
					ll.Error(err, "failed to save subchart package as tgz")
					err = fmt.Errorf("failed to save subchart package as tgz at location %s : %w", path, err)
					chartStatus.Error = err.Error()
					appGroup.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				err = r.RegistryClient.PushChart(ll, stagingRepoName, path, scc)
				if err != nil {
					ll.Error(err, "failed to push application subchart to staging registry")
					err = fmt.Errorf("failed to push application subchart to staging registry : %w", err)
					chartStatus.Error = err.Error()
					appGroup.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				chartStatus.Staged = true

				appGroup.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
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
			appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
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
			appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		appCh.Metadata.Name = utils.ConvertToDNS1123(appCh.Metadata.Name)

		_, err = registry.SaveChartPackage(appCh, stagingDir)
		if err != nil {
			ll.Error(err, "failed to save modified app chart to filesystem")
			err = fmt.Errorf("failed to save modified app chart to filesystem : %w", err)
			appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		// Replace existing chart with modified chart
		path := stagingDir + "/" + utils.ConvertToDNS1123(application.Spec.Chart.Name) + "-" + appCh.Metadata.Version + ".tgz"
		err = r.RegistryClient.PushChart(ll, r.StagingRepoName, path, appCh)
		defer func() {
			if r.CleanupDownloadedCharts {
				os.Remove(path)
			}
		}()
		if err != nil {
			ll.Error(err, "failed to push modified application chart to staging registry")
			err = fmt.Errorf("failed to push modified application chart to staging registry : %w", err)
			appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		appGroup.Status.Applications[i].ChartStatus.Staged = true
	}
	return nil
}

func (r *ApplicationGroupReconciler) reconcileDelete(ctx context.Context, appGroup *v1alpha1.ApplicationGroup, patch client.Patch) error {
	requeue := r.cleanupWorkflow(ctx, r.Log, appGroup)
	if requeue {
		// Change the app group spec into a reversing state
		appGroup.Reversing()
		if err := r.Status().Patch(ctx, appGroup, patch); err != nil {
			return err
		}
		r.Log.Info("reverse workflow is in progress")
	} else {
		r.Log.Info("cleaning up the applicationgroup resource")

		if _, ok := appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]; ok {
			appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation] = ""
		}
		controllerutil.RemoveFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		if err := r.Patch(ctx, appGroup, patch); err != nil {
			return err
		}
	}
	return nil
}
