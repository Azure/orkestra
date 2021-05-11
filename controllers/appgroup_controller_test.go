package controllers

import (
	"context"
	"time"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ApplicationGroup Controller", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Submit Bookinfo ApplicationGroup", func() {
		key := types.NamespacedName{
			Name:      "bookinfo",
			Namespace: "orkestra",
		}

		It("Should create successfully", func() {
			applicationGroup := bookinfo()

			Expect(k8sClient.Create(context.Background(), applicationGroup)).Should(Succeed())
			defer func() {
				g := &orkestrav1alpha1.ApplicationGroup{}
				k8sClient.Get(context.Background(), key, g)
				patch := client.MergeFrom(g.DeepCopy())
				g.Finalizers = nil
				Expect(k8sClient.Patch(context.Background(), g, patch)).Should(Succeed())
				Expect(k8sClient.Delete(context.Background(), g)).Should(Succeed())
			}()
			By("Submitting bookinfo")
			Eventually(func() error {
				g := &orkestrav1alpha1.ApplicationGroup{}
				return k8sClient.Get(context.Background(), key, g)
			}, timeout, interval).Should(BeNil())

			By("Expecting Finalizer to be set")
			Eventually(func() int {
				g := &orkestrav1alpha1.ApplicationGroup{}
				k8sClient.Get(context.Background(), key, g)
				return len(g.Finalizers)
			}, timeout, interval).Should(Not(BeZero()))

			By("Reading Status")
			Eventually(func() bool {
				g := &orkestrav1alpha1.ApplicationGroup{}
				k8sClient.Get(context.Background(), key, g)
				for _, condition := range g.Status.Conditions {
					switch condition.Type {
					case meta.ReadyCondition:
						if condition.Reason != meta.ProgressingReason {
							return false
						}
						if condition.Message != "workflow is reconciling..." {
							return false
						}
					case meta.DeployCondition:
						if condition.Reason != meta.SucceededReason {
							return false
						}
						if condition.Message != "application group reconciliation succeeded" {
							return false
						}
					// No other condition types are expected
					default:
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})
})
