package controllers

import (
	"context"
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplicationGroup Controller", func() {

	Context("ApplicationGroup", func() {
		var (
			namespace *corev1.Namespace
			ctx       context.Context
		)

		const (
			DefaultNamesapce = "orkestra"
		)

		BeforeEach(func() {
			// TODO: Namespace will be added once we have the namespace based support for ApplicationGroup
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "appgroup-test" + randStringRunes(5),
				},
			}
			ctx = context.Background()
			_ = k8sClient.Create(ctx, namespace)
			//Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should create Bookinfo spec successfully", func() {
			ctx := context.Background()
			applicationGroup := bookinfo()
			applicationGroup.Namespace = DefaultNamesapce
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, "application-group-finalizer")
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			helmReleaseList := &fluxhelmv2beta1.HelmReleaseList{}
			err = k8sClient.List(ctx, helmReleaseList)
			Expect(err).ToNot(HaveOccurred())
			oldHelmReleaseCount := len(helmReleaseList.Items)

			By("Making sure that the workflow goes into a running state")
			Eventually(func() bool {
				workflow := &v1alpha12.Workflow{}
				workflowKey := types.NamespacedName{Name: applicationGroup.Name, Namespace: applicationGroup.Namespace}
				_ = k8sClient.Get(ctx, workflowKey, workflow)
				return workflow.Status.Phase == v1alpha12.NodeRunning
			}, time.Minute, time.Second).Should(BeTrue())

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.SucceededReason
			}, time.Minute*4, time.Second).Should(BeTrue())

			By("checking that the all the HelmReleases have come up and are in a ready state")
			err = k8sClient.List(ctx, helmReleaseList)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(helmReleaseList.Items)).To(Equal(oldHelmReleaseCount + 6))
			allReady := true
			for _, release := range helmReleaseList.Items {
				if condition := meta.GetResourceCondition(&release, meta.ReadyCondition); condition.Reason == meta.SucceededReason {
					allReady = false
				}
			}
			Expect(allReady).To(BeTrue())

		})
	})
})
