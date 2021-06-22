package controllers_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	//"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers/testutils"
	"github.com/Azure/Orkestra/pkg/meta"

	//v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	//meta2 "github.com/fluxcd/pkg/apis/meta"
	"github.com/onsi/gomega"
	//"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	DefaultNamespace                 = "orkestra"
	DefaultTimeout                   = time.Minute * 5
	TotalHelmReleaseCount            = 6
	OnlyApplicationsHelmReleaseCount = 2
)

var (
	k8sClient client.Client

	BeNil        = gomega.BeNil
	BeTrue       = gomega.BeTrue
	Equal        = gomega.Equal
	HaveOccurred = gomega.HaveOccurred
)

func TestMain(m *testing.M) {
	var (
		testEnv *envtest.Environment
		err     error
	)

	// bootstrapping test environment
	k8sClient, testEnv, err = testutils.GetTestEnv()
	if err != nil {
		log.Fatal("error bootstrapping test environment, stopping...")
	}

	// run the tests
	exitCode := m.Run()

	// teardown the test environment
	testutils.CleanTestEnv(testEnv)

	os.Exit(exitCode)
}

func TestAppGroup_tc1(t *testing.T) {
	// Should create Bookinfo successfully
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	var err error
	var desc string
	ctx := context.Background()
	name := testutils.CreateAppGroupName("bookinfo")
	appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

	desc = "Getting the Helm Release list"
	hrList := &fluxhelmv2beta1.HelmReleaseList{}
	err = k8sClient.List(ctx, hrList, client.InNamespace(name))
	g.Expect(err).NotTo(HaveOccurred(), desc)
	oldHelmReleaseCount := len(hrList.Items)

	desc = "Making sure that the workflow goes into a running state"
	g.Eventually(func() bool {
		return testutils.IsWorkflowInRunningState(ctx, k8sClient, appGroup.Name, appGroup.Namespace)
	}, time.Minute, time.Second).Should(BeTrue(), desc)

	desc = "Waiting for the bookinfo object to reach a succeeded reason"
	g.Eventually(func() bool {
		return testutils.IsAppGroupInSucceededReason(ctx, k8sClient, appGroup)
	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)

	desc = "Checking that the all the HelmReleases have come up and are in a ready state"
	err = k8sClient.List(ctx, hrList, client.InNamespace(name))
	g.Expect(err).NotTo(HaveOccurred(), desc)
	g.Expect(len(hrList.Items)).To(Equal(oldHelmReleaseCount+TotalHelmReleaseCount), desc)
	allReady := true
	for _, release := range hrList.Items {
		condition := meta.GetResourceCondition(&release, meta.ReadyCondition)
		if condition.Reason == meta.SucceededReason {
			allReady = false
		}
	}
	g.Expect(allReady).To(BeTrue(), desc)

	desc = "Wait for all the HelmReleases to delete"
	err = k8sClient.Delete(ctx, appGroup)
	g.Expect(err).NotTo(HaveOccurred(), desc)

	desc = "Waiting for the Workflow to delete all the HelmReleases"
	g.Eventually(func() bool {
		helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
		if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
			return false
		}
		return len(helmReleases.Items) == 0
	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)
}

func TestAppGroup_tc2(t *testing.T) {
	// Should create only application releases with subchart nil successfully
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	var err error
	var desc string
	ctx := context.Background()
	name := testutils.CreateAppGroupName("bookinfo")
	appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
	for i := range appGroup.Spec.Applications {
		appGroup.Spec.Applications[i].Spec.Subcharts = nil
	}
	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

	desc = "Getting the Helm Release list"
	hrList := &fluxhelmv2beta1.HelmReleaseList{}
	err = k8sClient.List(ctx, hrList, client.InNamespace(name))
	g.Expect(err).NotTo(HaveOccurred(), desc)
	oldHelmReleaseCount := len(hrList.Items)

	desc = "Making sure that the workflow goes into a running state"
	g.Eventually(func() bool {
		return testutils.IsWorkflowInRunningState(ctx, k8sClient, appGroup.Name, appGroup.Namespace)
	}, time.Minute, time.Second).Should(BeTrue(), desc)

	desc = "Waiting for the bookinfo object to reach a succeeded reason"
	g.Eventually(func() bool {
		return testutils.IsAppGroupInSucceededReason(ctx, k8sClient, appGroup)
	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)

	desc = "Checking that the all the HelmReleases have come up and are in a ready state"
	err = k8sClient.List(ctx, hrList, client.InNamespace(name))
	g.Expect(err).NotTo(HaveOccurred(), desc)
	g.Expect(len(hrList.Items)).To(Equal(oldHelmReleaseCount+OnlyApplicationsHelmReleaseCount), desc)
	allReady := true
	for _, release := range hrList.Items {
		condition := meta.GetResourceCondition(&release, meta.ReadyCondition)
		if condition.Reason == meta.SucceededReason {
			allReady = false
		}
	}
	g.Expect(allReady).To(BeTrue(), desc)
}

func TestAppGroup_tc3(t *testing.T) {
	// Should fail to create and post a failed error state
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	ctx := context.Background()
	name := testutils.CreateAppGroupName("bookinfo")
	appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
	appGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"
	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

	g.Eventually(func() bool {
		return testutils.IsAppGroupInChartPullFailedReason(ctx, k8sClient, appGroup)
	}, time.Second*30, time.Second).Should(BeTrue())
}

// func TestAppGroup_tc4(t *testing.T) {
// 	// Should create the bookinfo and then update it
// 	t.Parallel()
// 	// g := gomega.NewGomegaWithT(t)

// 	// var err error
// 	// var desc string
// 	// ctx := context.Background()
// 	// name := testutils.CreateAppGroupName("bookinfo")
// 	// appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
// 	// testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

// 	// desc = "Waiting for the bookinfo object to reach a succeeded reason"
// 	// g.Eventually(func() bool {
// 	// 	return testutils.IsAppGroupInSucceededReason(ctx, k8sClient, appGroup)
// 	// }, DefaultTimeout, time.Second).Should(BeTrue(), desc)

// 	// desc = "Adding application to the AppGroup Spec after the AppGroup has fully reconciled"
// 	// newAppGroup := testutils.AddApplication(*appGroup, testutils.PodinfoApplication(name))
// 	// err = k8sClient.Update(ctx, &newAppGroup)
// 	// g.Expect(err).ToNot(HaveOccurred(), desc)

// 	// desc = "Waiting for the bookinfo application group to reach a succeeded reason"
// 	// g.Eventually(func() bool {
// 	// 	appGroup = &v1alpha1.ApplicationGroup{}
// 	// 	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(appGroup), appGroup); err != nil {
// 	// 		return false
// 	// 	}
// 	// 	return appGroup.GetReadyCondition() == meta.SucceededReason && appGroup.Generation == appGroup.Status.ObservedGeneration
// 	// }, time.Minute*2, time.Second).Should(BeTrue(), desc)
// }

// func TestAppGroup_tc5(t *testing.T) {
// 	// Should fail to install, then get updated and pass getting installed
// 	t.Parallel()
// 	g := gomega.NewGomegaWithT(t)

// 	var err error
// 	var desc string
// 	ctx := context.Background()
// 	name := testutils.CreateAppGroupName("bookinfo")
// 	appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
// 	appGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"
// 	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

// 	desc = "Waiting for the bookinfo object to go into a failed state"
// 	g.Eventually(func() bool {
// 		return testutils.IsAppGroupInChartPullFailedReason(ctx, k8sClient, appGroup)
// 	}, time.Second*30, time.Second).Should(BeTrue(), desc)

// 	desc = "Patching the bookinfo object with updated chart version"
// 	patch := client.MergeFrom(appGroup.DeepCopy())
// 	appGroup.Spec.Applications[0].Spec.Chart.Version = testutils.GetBookinfoChartVersion()
// 	err = k8sClient.Patch(ctx, appGroup, patch)
// 	g.Expect(err).ToNot(HaveOccurred(), desc)

// 	desc = "Waiting for the bookinfo object to reach a succeeded deploy condition"
// 	g.Eventually(func() bool {
// 		return testutils.IsAppGroupInProgressingReason(ctx, k8sClient, appGroup)
// 	}, time.Minute*2, time.Second).Should(BeTrue(), desc)
// }

// func TestAppGroup_tc6(t *testing.T) {
// 	// Should succeed to upgrade the versions of helm releases to newer versions
// 	t.Parallel()
// 	g := gomega.NewGomegaWithT(t)

// 	var err error
// 	var desc string
// 	ctx := context.Background()
// 	name := testutils.CreateAppGroupName("bookinfo")
// 	appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
// 	appGroup.Spec.Applications[1].Spec.Chart.Version = testutils.GetAmbassadorOldChartVersion()

// 	key := client.ObjectKeyFromObject(appGroup)
// 	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

// 	desc = "Waiting for the bookinfo object to reach a succeeded reason"
// 	g.Eventually(func() bool {
// 		return testutils.IsAppGroupInSucceededReason(ctx, k8sClient, appGroup)
// 	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)

// 	desc = "Upgrading the charts to a newer version"
// 	patch := client.MergeFrom(appGroup.DeepCopy())
// 	appGroup.Spec.Applications[1].Spec.Chart.Version = testutils.GetAmbassadorChartVersion()
// 	err = k8sClient.Patch(ctx, appGroup, patch)
// 	g.Expect(err).ToNot(HaveOccurred(), desc)

// 	desc = "Waiting for the application group to start its upgrade"
// 	g.Eventually(func() bool {
// 		appGroup = &v1alpha1.ApplicationGroup{}
// 		if err := k8sClient.Get(ctx, key, appGroup); err != nil {
// 			return false
// 		}
// 		return appGroup.GetReadyCondition() == meta.ProgressingReason && appGroup.Generation > 1
// 	}, time.Second*30, time.Second).Should(BeTrue(), desc)

// 	desc = "Waiting for the newer version of the charts to be released"
// 	g.Eventually(func() bool {
// 		hr := &fluxhelmv2beta1.HelmRelease{}
// 		err := k8sClient.Get(ctx, types.NamespacedName{
// 			Name:      "ambassador",
// 			Namespace: name,
// 		}, hr)
// 		if err != nil {
// 			return false
// 		}

// 		appGroup = &v1alpha1.ApplicationGroup{}
// 		if err := k8sClient.Get(ctx, key, appGroup); err != nil {
// 			return false
// 		}
// 		return hr.Spec.Chart.Spec.Version == testutils.GetAmbassadorChartVersion() && appGroup.GetReadyCondition() == meta.SucceededReason
// 	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)
// }

// func TestAppGroup_tc7(t *testing.T) {
// 	// Should succeed to rollback helm chart versions on failure
// 	t.Parallel()
// 	g := gomega.NewGomegaWithT(t)

// 	var err error
// 	var desc string
// 	ctx := context.Background()
// 	name := testutils.CreateAppGroupName("bookinfo")
// 	appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
// 	appGroup.Spec.Applications[1].Spec.Chart.Version = testutils.GetAmbassadorOldChartVersion()
// 	key := client.ObjectKeyFromObject(appGroup)
// 	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

// 	desc = "Waiting for the bookinfo object to reach a succeeded reason"
// 	g.Eventually(func() bool {
// 		appGroup = &v1alpha1.ApplicationGroup{}
// 		if err := k8sClient.Get(ctx, key, appGroup); err != nil {
// 			return false
// 		}
// 		_, exist := appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]
// 		return appGroup.GetReadyCondition() == meta.SucceededReason && exist
// 	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)

// 	desc = "Upgrading the ambassador chart to a newer version while intentionally timing out the last DAG step"
// 	patch := client.MergeFrom(appGroup.DeepCopy())
// 	appGroup.Spec.Applications[1].Spec.Chart.Version = testutils.GetAmbassadorChartVersion()
// 	appGroup.Spec.Applications[0].Spec.Release.Timeout = &metav1.Duration{Duration: time.Second}
// 	err = k8sClient.Patch(ctx, appGroup, patch)
// 	g.Expect(err).ToNot(HaveOccurred(), desc)

// 	desc = "Waiting for the application group to start its upgrade"
// 	g.Eventually(func() bool {
// 		if err := k8sClient.Get(ctx, key, appGroup); err != nil {
// 			return false
// 		}
// 		return appGroup.GetReadyCondition() == meta.ProgressingReason && appGroup.Generation > 1
// 	}, time.Second*30, time.Second).Should(BeTrue(), desc)

// 	desc = "Waiting for the newer version of the charts to be released"
// 	g.Eventually(func() bool {
// 		hr := &fluxhelmv2beta1.HelmRelease{}
// 		err := k8sClient.Get(ctx, types.NamespacedName{
// 			Name:      "ambassador",
// 			Namespace: name,
// 		}, hr)
// 		if err != nil {
// 			return false
// 		}

// 		return hr.Spec.Chart.Spec.Version == testutils.GetAmbassadorChartVersion() && meta.GetResourceCondition(hr, meta.ReadyCondition).Reason == meta2.ReconciliationSucceededReason
// 	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)

// 	desc = "Ensuring that the applications rollback to their starting version"
// 	g.Eventually(func() bool {
// 		hr := &fluxhelmv2beta1.HelmRelease{}
// 		err := k8sClient.Get(ctx, types.NamespacedName{
// 			Name:      "ambassador",
// 			Namespace: name,
// 		}, hr)
// 		if err != nil {
// 			return false
// 		}
// 		appGroup = &v1alpha1.ApplicationGroup{}
// 		if err := k8sClient.Get(ctx, key, appGroup); err != nil {
// 			return false
// 		}

// 		return hr.Spec.Chart.Spec.Version == testutils.GetAmbassadorOldChartVersion() && meta.GetResourceCondition(hr, meta.ReadyCondition).Reason == meta2.ReconciliationSucceededReason && appGroup.GetReadyCondition() == meta.WorkflowFailedReason && appGroup.GetWorkflowCondition(v1alpha1.Rollback) == meta.SucceededReason
// 	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)
// }

// func TestAppGroup_tc8(t *testing.T) {
// 	// Should create the bookinfo and then delete it while in progress
// 	t.Parallel()
// 	// g := gomega.NewGomegaWithT(t)

// 	// var err error
// 	// var desc string
// 	// ctx := context.Background()
// 	// name := testutils.CreateAppGroupName("bookinfo")
// 	// appGroup := testutils.DefaultAppGroup(name, DefaultNamespace, name)
// 	// testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

// 	// desc = "Making sure that the workflow goes into a running state"
// 	// g.Eventually(func() bool {
// 	// 	return testutils.IsWorkflowInRunningState(ctx, k8sClient, appGroup.Name, appGroup.Namespace)
// 	// }, time.Minute, time.Second).Should(BeTrue(), desc)

// 	// desc = "Waiting for the bookinfo object to reach a progressing reason"
// 	// g.Eventually(func() bool {
// 	// 	return testutils.IsAppGroupInProgressingReason(ctx, k8sClient, appGroup)
// 	// }, time.Minute, time.Second).Should(BeTrue(), desc)

// 	// desc = "Waiting for the ambassador helm release to be ready"
// 	// g.Eventually(func() bool {
// 	// 	hr := &fluxhelmv2beta1.HelmRelease{}
// 	// 	err := k8sClient.Get(ctx, types.NamespacedName{
// 	// 		Name:      "ambassador",
// 	// 		Namespace: name,
// 	// 	}, hr)
// 	// 	if err != nil {
// 	// 		return false
// 	// 	}
// 	// 	readyCondition := meta.GetResourceCondition(hr, meta.ReadyCondition)
// 	// 	if readyCondition == nil {
// 	// 		return false
// 	// 	}
// 	// 	return readyCondition.Reason == meta2.ReconciliationSucceededReason
// 	// }, time.Minute*2, time.Second).Should(BeTrue(), desc)

// 	// // Wait for all the HelmReleases to delete
// 	// err = k8sClient.Delete(ctx, appGroup, client.PropagationPolicy(metav1.DeletePropagationForeground))
// 	// g.Expect(err).NotTo(HaveOccurred())

// 	// desc = "Making sure that the workflow goes into a suspended state"
// 	// g.Eventually(func() bool {
// 	// 	return testutils.IsWorkflowInSuspendedState(ctx, k8sClient, name, appGroup.Namespace)
// 	// }, time.Minute, time.Second).Should(BeTrue(), desc)

// 	// desc = "Waiting for the Workflow to delete all the HelmReleases"
// 	// g.Eventually(func() bool {
// 	// 	hr := &fluxhelmv2beta1.HelmReleaseList{}
// 	// 	if err := k8sClient.List(ctx, hr, client.InNamespace(name)); err != nil {
// 	// 		return false
// 	// 	}
// 	// 	return len(hr.Items) == 0
// 	// }, time.Minute*3, time.Second).Should(BeTrue(), desc)

// 	// desc = "Waiting for all the Workflows to be cleaned up from the cluster"
// 	// g.Eventually(func() bool {
// 	// 	workflowList := &v1alpha13.WorkflowList{}
// 	// 	if err := k8sClient.List(ctx, workflowList, client.InNamespace(name)); err != nil {
// 	// 		return false
// 	// 	}
// 	// 	return len(workflowList.Items) == 0
// 	// }, time.Minute, time.Second).Should(BeTrue(), desc)
// }

// func TestAppGroup_tc9(t *testing.T) {
// 	// Should delete the application group if reverse workflow is removed
// 	t.Parallel()
// 	g := gomega.NewGomegaWithT(t)

// 	var err error
// 	var desc string
// 	ctx := context.Background()
// 	name := testutils.CreateAppGroupName("bookinfo")
// 	appGroup := testutils.SmallAppGroup(name, DefaultNamespace, name)
// 	key := client.ObjectKeyFromObject(appGroup)
// 	testutils.ApplyObjToK8sAndRegisterCleanup(ctx, t, k8sClient, appGroup)

// 	desc = "Waiting for the bookinfo object to reach a succeeded reason"
// 	g.Eventually(func() bool {
// 		return testutils.IsAppGroupInSucceededReason(ctx, k8sClient, appGroup)
// 	}, DefaultTimeout, time.Second).Should(BeTrue(), desc)

// 	desc = "Deleting the application group and deleting the workflow"
// 	err = k8sClient.Delete(ctx, appGroup)
// 	g.Expect(err).To(BeNil(), desc)

// 	wf := &v1alpha13.Workflow{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: DefaultNamespace,
// 		},
// 	}
// 	err = k8sClient.Delete(ctx, wf)
// 	g.Expect(err).To(BeNil(), desc)

// 	g.Eventually(func() bool {
// 		appGroup = &v1alpha1.ApplicationGroup{}
// 		return errors.IsNotFound(k8sClient.Get(ctx, key, appGroup))
// 	}, time.Second*30, time.Second).Should(BeTrue(), desc)
// }
