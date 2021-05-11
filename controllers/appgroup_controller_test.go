package controllers

import (
	"context"
	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplicationGroup Controller", func() {

	Context("ApplicationGroup", func() {
		var (
			namespace *corev1.Namespace
		)

		BeforeEach(func() {
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "appgroup-test" + randStringRunes(5),
				},
			}
			err := k8sClient.Create(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should create Bookinfo spec successfully", func() {
			ctx := context.Background()
			applicationGroup := bookinfo()
			applicationGroup.Namespace = namespace.Name
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			Expect(k8sClient.Create(ctx, applicationGroup)).To(Succeed())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				k8sClient.Delete(ctx, applicationGroup)
			}()

			helmReleaseList := &fluxhelmv2beta1.HelmReleaseList{}
			err := k8sClient.List(ctx, helmReleaseList)
			Expect(err).ToNot(HaveOccurred())
			oldHelmReleaseCount := len(helmReleaseList.Items)

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				temp := &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, temp); err != nil {
					return false
				}
				return temp.GetReadyCondition() == meta.SucceededReason
			}, time.Minute*3, time.Second).Should(BeTrue())

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
