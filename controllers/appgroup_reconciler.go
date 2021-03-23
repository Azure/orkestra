package controllers

import (
	"context"
	"errors"
	"os"
	"strings"

	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/workflow"
	"github.com/go-logr/logr"
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

func (r *ApplicationGroupReconciler) reconcile(ctx context.Context, l logr.Logger, ns string, appGroup *orkestrav1alpha1.ApplicationGroup) (bool, error) {
	l = l.WithValues(appgroupNameKey, appGroup.Name)
	l.V(3).Info("Reconciling ApplicationGroup object")

	if len(appGroup.Spec.Applications) == 0 {
		l.Error(ErrInvalidSpec, "ApplicationGroup must list atleast one Application")
		err := fmt.Errorf("application group must list atleast one Application : %w", ErrInvalidSpec)
		appGroup.Status.Error = err.Error()
		return false, err
	}

	err := r.reconcileApplications(l, appGroup)
	if err != nil {
		l.Error(err, "failed to reconcile the applications")
		err = fmt.Errorf("failed to reconcile the applications : %w", err)
		return false, err
	}
	// if target workflow namespace is unset, then set it to the default namespace explicitly
	if ns == "" {
		ns = defaultNamespace()
	}

	// Generate the Workflow object to submit to Argo
	return r.generateWorkflow(ctx, l, ns, appGroup)
}

func (r *ApplicationGroupReconciler) reconcileApplications(l logr.Logger, appGroup *v1alpha1.ApplicationGroup) error {
	stagingDir := r.TargetDir + "/" + r.StagingRepoName
	// Pull and conditionally stage application & dependency charts
	for i, application := range appGroup.Spec.Applications {
		ll := l.WithValues("application", application.Name)
		ll.V(3).Info("performing chart actions")

		if appGroup.Status.Applications[i].Subcharts == nil {
			appGroup.Status.Applications[i].Subcharts = make(map[string]orkestrav1alpha1.ChartStatus)
		}

		repoKey := application.Spec.ChartRepoNickname
		repoPath := application.Spec.RepoPath
		name := application.Spec.HelmReleaseSpec.Name
		version := application.Spec.HelmReleaseSpec.Version

		fpath, appCh, err := r.RegistryClient.PullChart(ll, repoKey, repoPath, name, version)
		defer func() {
			if r.Cfg.CleanupDownloadedCharts {
				os.Remove(fpath)
			}
		}()
		if err != nil || appCh == nil {
			err = fmt.Errorf("failed to pull application chart %s/%s:%s : %w", repoKey, name, version, err)
			appGroup.Status.Error = err.Error()
			ll.Error(err, "failed to pull application chart")
			return err
		}

		if appCh.Dependencies() != nil {
			for _, sc := range appCh.Dependencies() {
				cs := orkestrav1alpha1.ChartStatus{
					Version: sc.Metadata.Version,
				}
				appGroup.Status.Applications[i].Subcharts[sc.Name()] = cs
			}

			stagingRepoName := r.StagingRepoName
			// If Dependencies - extract subchart and push each to staging registry
			if isDependenciesEmbedded(appCh) {
				for _, sc := range appCh.Dependencies() {
					cs := orkestrav1alpha1.ChartStatus{
						Version: sc.Metadata.Version,
					}

					if err := sc.Validate(); err != nil {
						ll.Error(err, "failed to validate application subchart for staging registry")
						err = fmt.Errorf("failed to validate application subchart for staging registry : %w", err)
						cs.Error = err.Error()
						appGroup.Status.Applications[i].Subcharts[sc.Name()] = cs
						appGroup.Status.Error = cs.Error
						return err
					}

					path, err := registry.SaveChartPackage(sc, stagingDir)
					if err != nil {
						ll.Error(err, "failed to save subchart package as tgz")
						err = fmt.Errorf("failed to save subchart package as tgz at location %s : %w", path, err)
						cs.Error = err.Error()
						appGroup.Status.Applications[i].Subcharts[sc.Name()] = cs
						appGroup.Status.Error = cs.Error
						return err
					}

					err = r.RegistryClient.PushChart(ll, stagingRepoName, path, sc)
					if err != nil {
						ll.Error(err, "failed to push application subchart to staging registry")
						err = fmt.Errorf("failed to push application subchart to staging registry : %w", err)
						cs.Error = err.Error()
						appGroup.Status.Applications[i].Subcharts[sc.Name()] = cs
						appGroup.Status.Error = cs.Error
						return err
					}

					cs.Staged = true
					cs.Version = sc.Metadata.Version
					cs.Error = ""

					appGroup.Status.Applications[i].Subcharts[sc.Name()] = cs
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
				appGroup.Status.Error = err.Error()
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
				appGroup.Status.Error = err.Error()
				appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
				return err
			}

			_, err = registry.SaveChartPackage(appCh, stagingDir)
			if err != nil {
				ll.Error(err, "failed to save modified app chart to filesystem")
				err = fmt.Errorf("failed to save modified app chart to filesystem : %w", err)
				appGroup.Status.Error = err.Error()
				appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
				return err
			}

			// Replace existing chart with modified chart
			path := stagingDir + "/" + application.Spec.HelmReleaseSpec.Name + "-" + appCh.Metadata.Version + ".tgz"
			err = r.RegistryClient.PushChart(ll, stagingRepoName, path, appCh)
			defer func() {
				if r.Cfg.CleanupDownloadedCharts {
					os.Remove(path)
				}
			}()
			if err != nil {
				ll.Error(err, "failed to push modified application chart to staging registry")
				err = fmt.Errorf("failed to push modified application chart to staging registry : %w", err)
				appGroup.Status.Applications[i].ChartStatus.Error = err.Error()
				appGroup.Status.Error = err.Error()
				return err
			}

			appGroup.Status.Applications[i].ChartStatus.Staged = true
		}
	}
	return nil
}

func (r *ApplicationGroupReconciler) generateWorkflow(ctx context.Context, logr logr.Logger, ns string, g *orkestrav1alpha1.ApplicationGroup) (requeue bool, err error) {
	err = r.Engine.Generate(ctx, logr, ns, g)
	if err != nil {
		logr.Error(err, "engine failed to generate workflow")
		return false, fmt.Errorf("failed to generate workflow : %w", err)
	}

	err = r.Engine.Submit(ctx, logr, g)
	if err != nil {
		if errors.Is(err, workflow.ErrNamespaceTerminating) {
			logr.V(1).Info("namespace is in terminating state")
			return true, err
		}
		logr.Error(err, "engine failed to submit workflow")
		return false, err
	}

	g.Status.Phase = orkestrav1alpha1.Init

	return true, nil
}

func defaultNamespace() string {
	if ns, ok := os.LookupEnv("WORKFLOW_NAMESPACE"); ok {
		return ns
	}
	return "orkestra"
}

func templatesContainsYAML(ch *chart.Chart) (bool, error) {
	if ch == nil {
		return false, fmt.Errorf("chart cannot be nil")
	}

	for _, f := range ch.Templates {
		if strings.Contains(f.Name, ".yaml") {
			return true, nil
		}
	}
	return false, nil
}
