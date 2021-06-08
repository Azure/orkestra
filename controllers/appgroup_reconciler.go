package controllers

import (
	"context"
	"os"
	"strings"

	"github.com/Azure/Orkestra/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
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

func (r *ApplicationGroupReconciler) reconcile(ctx context.Context, l logr.Logger, appGroup *v1alpha1.ApplicationGroup) (bool, error) {
	l = l.WithValues(v1alpha1.AppGroupNameKey, appGroup.Name)
	l.V(3).Info("Reconciling ApplicationGroup object")

	if len(appGroup.Spec.Applications) == 0 {
		l.Error(ErrInvalidSpec, "ApplicationGroup must list atleast one Application")
		err := fmt.Errorf("application group must list atleast one Application : %w", ErrInvalidSpec)
		return false, err
	}

	err := r.reconcileApplications(l, appGroup)
	if err != nil {
		l.Error(err, "failed to reconcile the applications")
		err = fmt.Errorf("failed to reconcile the applications : %w", err)
		return false, err
	}

	// Generate the Workflow object to submit to Argo
	return r.generateWorkflow(ctx, l, appGroup)
}

func (r *ApplicationGroupReconciler) reconcileApplications(l logr.Logger, appGroup *v1alpha1.ApplicationGroup) error {
	stagingDir := r.TargetDir + "/" + r.StagingRepoName

	// Init the application status every time we re-reconcile the applications
	initAppStatus(appGroup)

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
					if !isFileYAML(f.Name) {
						t := &chart.File{}
						_ = copier.Copy(t, f)
						t.Name = addAppChartNameToFile(t.Name, appCh.Name())
						scc.Templates = append(scc.Templates, t)
					}
				}

				hashAppChName := utils.GetHash(appCh.Metadata.Name)
				scc.Metadata.Name = utils.ConvertToDNS1123(utils.JoinForDNS1123(hashAppChName, scc.Metadata.Name))
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

		templateHasYAML, err := templatesContainsYAML(appCh)
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

func (r *ApplicationGroupReconciler) reconcileDelete(ctx context.Context, appGroup v1alpha1.ApplicationGroup, patch client.Patch) (ctrl.Result, error) {
	requeue := r.cleanupWorkflow(ctx, r.Log, appGroup)
	if requeue {
		// Change the app group spec into a reversing state
		appGroup.Reversing()
		_ = r.Status().Patch(ctx, &appGroup, patch)

		r.Log.Info("reverse workflow is in progress")
	} else {
		r.Log.Info("cleaning up the applicationgroup resource")

		if _, ok := appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]; ok {
			appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation] = ""
		}
		controllerutil.RemoveFinalizer(&appGroup, v1alpha1.AppGroupFinalizer)
		_ = r.Patch(ctx, &appGroup, patch)
	}
	return r.handleResponseAndEvent(ctx, r.Log, appGroup, patch, requeue, nil)
}

func initAppStatus(appGroup *v1alpha1.ApplicationGroup) {
	// Initialize the Status fields if not already setup
	appGroup.Status.Applications = make([]v1alpha1.ApplicationStatus, 0, len(appGroup.Spec.Applications))
	for _, app := range appGroup.Spec.Applications {
		status := v1alpha1.ApplicationStatus{
			Name:        app.Name,
			ChartStatus: v1alpha1.ChartStatus{Version: app.Spec.Chart.Version},
			Subcharts:   make(map[string]v1alpha1.ChartStatus),
		}
		appGroup.Status.Applications = append(appGroup.Status.Applications, status)
	}
}

func (r *ApplicationGroupReconciler) generateWorkflow(ctx context.Context, logr logr.Logger, g *v1alpha1.ApplicationGroup) (requeue bool, err error) {
	err = r.Engine.Generate(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to generate workflow")
		return false, fmt.Errorf("failed to generate workflow : %w", err)
	}

	err = r.Engine.Submit(ctx, logr, g)
	if err != nil {
		logr.Error(err, "engine failed to submit workflow")
		return false, err
	}
	return true, nil
}

func templatesContainsYAML(ch *chart.Chart) (bool, error) {
	if ch == nil {
		return false, fmt.Errorf("chart cannot be nil")
	}

	for _, f := range ch.Templates {
		if isFileYAML(f.Name) {
			return true, nil
		}
	}
	return false, nil
}

func isFileYAML(f string) bool {
	f = strings.ToLower(f)
	if strings.HasSuffix(f, "yml") || strings.HasSuffix(f, "yaml") {
		return true
	}
	return false
}

func addAppChartNameToFile(name, a string) string {
	prefix := "templates/"
	name = strings.TrimPrefix(name, prefix)
	name = a + "_" + name
	return prefix + name
}
