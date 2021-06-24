package controllers_test

import (
	"context"
	"time"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers/testutils"
	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("ApplicationGroup Controller", func() {
	const (
		DefaultNamespace                 = "orkestra"
		DefaultTimeout                   = time.Minute * 5
		TotalHelmReleaseCount            = 6
		OnlyApplicationsHelmReleaseCount = 2
	)

	var (
		ctx context.Context
		err error

		name     string
		appGroup *v1alpha1.ApplicationGroup
		key      types.NamespacedName
	)

	BeforeEach(func() {
		ctx = context.Background()
		err = nil

		name = testutils.CreateUniqueAppGroupName("bookinfo")
		appGroup = testutils.DefaultAppGroup(name, DefaultNamespace, name)
		key = client.ObjectKeyFromObject(appGroup)
	})

	AfterEach(func() {
		By("Cleanup: Deleting the bookinfo object from the cluster")
		patch := client.MergeFrom(appGroup.DeepCopy())
		controllerutil.RemoveFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		_ = k8sClient.Patch(ctx, appGroup, patch)
		_ = k8sClient.Delete(ctx, appGroup)

		By("Cleanup: Calling delete on the Helm Releases")
		_ = k8sClient.DeleteAllOf(ctx, &fluxhelmv2beta1.HelmRelease{}, client.InNamespace(name))
		_ = k8sClient.DeleteAllOf(ctx, &v1alpha13.Workflow{}, client.InNamespace(name))
	})

	It("Should create Bookinfo spec successfully", func() {
		By("Applying the bookinfo object to the cluster")
		err = k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Getting the Helm Release list")
		hrList := &fluxhelmv2beta1.HelmReleaseList{}
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).ToNot(HaveOccurred())
		oldHelmReleaseCount := len(hrList.Items)

		By("Making sure that the workflow goes into a running state")
		Eventually(func() bool {
			workflow := &v1alpha13.Workflow{}
			workflowKey := types.NamespacedName{Name: appGroup.Name, Namespace: appGroup.Namespace}
			_ = k8sClient.Get(ctx, workflowKey, workflow)
			return string(workflow.Status.Phase) == string(v1alpha13.NodeRunning)
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			appGroup = &v1alpha1.ApplicationGroup{}
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, DefaultTimeout, time.Second).Should(BeTrue())

		By("Checking that the all the HelmReleases have come up and are in a ready state")
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(hrList.Items)).To(Equal(oldHelmReleaseCount + TotalHelmReleaseCount))
		allReady := true
		for _, release := range hrList.Items {
			condition := meta.GetResourceCondition(&release, meta.ReadyCondition)
			if condition.Reason == meta.SucceededReason {
				allReady = false
			}
		}
		Expect(allReady).To(BeTrue())

		By("Waiting for all the HelmReleases to delete")
		err = k8sClient.Delete(ctx, appGroup)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the Workflow to delete all the HelmReleases")
		Eventually(func() bool {
			helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
			if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
				return false
			}
			return len(helmReleases.Items) == 0
		}, DefaultTimeout, time.Second).Should(BeTrue())
	})

	It("Should create only application releases with subchart nil successfully", func() {
		for i := range appGroup.Spec.Applications {
			appGroup.Spec.Applications[i].Spec.Subcharts = nil
		}

		By("Applying the bookinfo object to the cluster")
		err = k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Getting the Helm Release list")
		hrList := &fluxhelmv2beta1.HelmReleaseList{}
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).ToNot(HaveOccurred())
		oldHelmReleaseCount := len(hrList.Items)

		By("Making sure that the workflow goes into a running state")
		Eventually(func() bool {
			workflow := &v1alpha13.Workflow{}
			workflowKey := types.NamespacedName{Name: appGroup.Name, Namespace: appGroup.Namespace}
			_ = k8sClient.Get(ctx, workflowKey, workflow)
			return string(workflow.Status.Phase) == string(v1alpha13.NodeRunning)
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			appGroup = &v1alpha1.ApplicationGroup{}
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, DefaultTimeout, time.Second).Should(BeTrue())

		By("Checking that the all the HelmReleases have come up and are in a ready state")
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(hrList.Items)).To(Equal(oldHelmReleaseCount + OnlyApplicationsHelmReleaseCount))
		allReady := true
		for _, release := range hrList.Items {
			condition := meta.GetResourceCondition(&release, meta.ReadyCondition)
			if condition.Reason == meta.SucceededReason {
				allReady = false
			}
		}
		Expect(allReady).To(BeTrue())
	})

	It("Should fail to create and post a failed error state", func() {
		appGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"

		By("Applying the bookinfo object to the cluster")
		err = k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a chart pull failed reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			readyCondition := meta.GetResourceCondition(appGroup, meta.ReadyCondition)
			if readyCondition == nil {
				return false
			}
			return readyCondition.Reason == meta.ChartPullFailedReason
		}, time.Second*30, time.Second).Should(BeTrue())
	})

	It("Should create the bookinfo and then update it", func() {
		By("Applying the bookinfo object to the cluster")
		err = k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			appGroup = &v1alpha1.ApplicationGroup{}
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, DefaultTimeout, time.Second).Should(BeTrue())

		By("Adding application to the AppGroup Spec after the AppGroup has fully reconciled")
		newAppGroup := testutils.AddApplication(*appGroup, testutils.PodinfoApplication(name))
		err = k8sClient.Update(ctx, &newAppGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo application group to reach a succeeded reason")
		Eventually(func() bool {
			appGroup = &v1alpha1.ApplicationGroup{}
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason && appGroup.Generation == appGroup.Status.ObservedGeneration
		}, DefaultTimeout*2, time.Second).Should(BeTrue())
	})
})
