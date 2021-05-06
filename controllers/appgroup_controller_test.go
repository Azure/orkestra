package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplicationGroup Controller", func() {
	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Submit Bookinfo ApplicationGroup", func() {
		It("Should create successfully", func() {
			// applicationGroupKey := types.NamespacedName{
			// 	Name: "bookinfo",
			// }

			applicationGroup := bookinfo()

			By("Applying the bookinfo object to the cluster")
			Expect(k8sClient.Create(context.Background(), applicationGroup)).Should(Succeed())
			time.Sleep(time.Second * 5)
			defer func() {
				Expect(k8sClient.Delete(context.Background(), applicationGroup)).Should(Succeed())
				time.Sleep(time.Second * 5)
			}()
		})
	})
})
