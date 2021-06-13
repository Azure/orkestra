package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	meta2 "github.com/fluxcd/pkg/apis/meta"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplicationGroup Controller", func() {

	Context("ApplicationGroup", func() {
		var (
			namespace *corev1.Namespace
			name      string
			ctx       context.Context
		)

		const (
			DefaultNamespace      = "orkestra"
			DefaultTimeout        = time.Minute * 5
			TotalHelmReleaseCount = 6
		)

		BeforeEach(func() {
			ctx = context.Background()
			_ = k8sClient.Create(ctx, namespace)
			//Expect(err).ToNot(HaveOccurred())

			// Give the application group a unique name
			name = "bookinfo-" + randStringRunes(6)
		})

		AfterEach(func() {
			// Call delete on the HelmReleases for cleanup
			_ = k8sClient.DeleteAllOf(ctx, &fluxhelmv2beta1.HelmRelease{}, client.InNamespace(name))
			_ = k8sClient.DeleteAllOf(ctx, &v1alpha12.Workflow{}, client.InNamespace(name))
		})

		It("Should create Bookinfo spec successfully", func() {
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Name = name
			applicationGroup.Namespace = DefaultNamespace
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			helmReleaseList := &fluxhelmv2beta1.HelmReleaseList{}
			err = k8sClient.List(ctx, helmReleaseList, client.InNamespace(name))
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
			}, DefaultTimeout, time.Second).Should(BeTrue())

			By("checking that the all the HelmReleases have come up and are in a ready state")
			err = k8sClient.List(ctx, helmReleaseList, client.InNamespace(name))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(helmReleaseList.Items)).To(Equal(oldHelmReleaseCount + TotalHelmReleaseCount))
			allReady := true
			for _, release := range helmReleaseList.Items {
				if condition := meta.GetResourceCondition(&release, meta.ReadyCondition); condition.Reason == meta.SucceededReason {
					allReady = false
				}
			}
			Expect(allReady).To(BeTrue())

			// Wait for all the HelmReleases to delete
			err = k8sClient.Delete(ctx, applicationGroup)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the Workflow to delete all the HelmReleases")
			Eventually(func() bool {
				helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
				if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
					return false
				}
				return len(helmReleases.Items) == 0
			}, DefaultTimeout, time.Second).Should(BeTrue())

		})

		It("should fail to create and post a failed error state", func() {
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Namespace = DefaultNamespace

			applicationGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			Eventually(func() bool {
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				readyCondition := meta.GetResourceCondition(applicationGroup, meta.ReadyCondition)
				if readyCondition == nil {
					return false
				}
				return readyCondition.Reason == meta.FailedReason
			}, time.Second*30, time.Second).Should(BeTrue())
		})

		It("should create the bookinfo spec and then update it", func() {
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Namespace = DefaultNamespace
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.SucceededReason
			}, DefaultTimeout, time.Second).Should(BeTrue())

			By("Adding an Application to the ApplicationGroup Spec after the ApplicationGroup has fully reconciled")
			newAppGroup := AddApplication(*applicationGroup, podinfoApplication(name))
			err = k8sClient.Update(ctx, &newAppGroup)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the bookinfo application group to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.SucceededReason &&
					applicationGroup.Generation == applicationGroup.Status.ObservedGeneration
			}, time.Minute*2, time.Second).Should(BeTrue())
		})

		It("should fail to install, then get updated and pass getting installed", func() {
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Namespace = DefaultNamespace

			applicationGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			By("Waiting for the bookinfo object to go into a failed state")
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(applicationGroup), applicationGroup); err != nil {
					return false
				}
				readyCondition := meta.GetResourceCondition(applicationGroup, meta.ReadyCondition)
				if readyCondition == nil {
					return false
				}
				return readyCondition.Reason == meta.FailedReason
			}, time.Second*30, time.Second).Should(BeTrue())

			patch := client.MergeFrom(applicationGroup.DeepCopy())
			applicationGroup.Spec.Applications[0].Spec.Chart.Version = bookinfoChartVersion
			err = k8sClient.Patch(ctx, applicationGroup, patch)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the bookinfo object to reach a succeeded deploy condition")
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(applicationGroup), applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.ProgressingReason
			}, time.Minute*2, time.Second).Should(BeTrue())
		})

		It("should succeed to upgrade the versions of helm releases to newer versions", func() {
			By("creating three releases that use older versions of charts")
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Namespace = DefaultNamespace
			applicationGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorOldChartVersion
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.SucceededReason
			}, DefaultTimeout, time.Second).Should(BeTrue())

			By("upgrading the charts to a newer version")
			patch := client.MergeFrom(applicationGroup.DeepCopy())
			applicationGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorChartVersion
			err = k8sClient.Patch(ctx, applicationGroup, patch)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the application group to start its upgrade")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.ProgressingReason && applicationGroup.Generation > 1
			}, time.Second*30, time.Second).Should(BeTrue())

			By("waiting for the newer version of the charts to be released")
			Eventually(func() bool {
				ambassadorHelmRelease := &fluxhelmv2beta1.HelmRelease{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: ambassador, Namespace: name}, ambassadorHelmRelease); err != nil {
					return false
				}
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return ambassadorHelmRelease.Spec.Chart.Spec.Version == ambassadorChartVersion &&
					applicationGroup.GetReadyCondition() == meta.SucceededReason
			}, DefaultTimeout, time.Second).Should(BeTrue())
		})

		FIt("should succeed to rollback helm chart versions on failure", func() {
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Namespace = DefaultNamespace
			applicationGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorOldChartVersion
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				_, exist := applicationGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]
				return applicationGroup.GetReadyCondition() == meta.SucceededReason && exist

			}, DefaultTimeout, time.Second).Should(BeTrue())

			By("upgrading the ambassador chart to a newer version while intentionally timing out the last DAG step")
			patch := client.MergeFrom(applicationGroup.DeepCopy())
			applicationGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorChartVersion
			applicationGroup.Spec.Applications[0].Spec.Release.Timeout = &metav1.Duration{Duration: time.Second}
			err = k8sClient.Patch(ctx, applicationGroup, patch)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the application group to start its upgrade")
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.ProgressingReason && applicationGroup.Generation > 1
			}, time.Second*30, time.Second).Should(BeTrue())

			By("waiting for the newer version of the charts to be released")
			Eventually(func() bool {
				ambassadorHelmRelease := &fluxhelmv2beta1.HelmRelease{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: ambassador, Namespace: name}, ambassadorHelmRelease); err != nil {
					return false
				}
				return ambassadorHelmRelease.Spec.Chart.Spec.Version == ambassadorChartVersion &&
					meta.GetResourceCondition(ambassadorHelmRelease, meta.ReadyCondition).Reason == meta2.ReconciliationSucceededReason
			}, DefaultTimeout, time.Second).Should(BeTrue())

			By("ensuring that the applications rollback to their starting version")
			Eventually(func() bool {
				ambassadorHelmRelease := &fluxhelmv2beta1.HelmRelease{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: ambassador, Namespace: name}, ambassadorHelmRelease); err != nil {
					return false
				}
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return ambassadorHelmRelease.Spec.Chart.Spec.Version == ambassadorOldChartVersion &&
					meta.GetResourceCondition(ambassadorHelmRelease, meta.ReadyCondition).Reason == meta2.ReconciliationSucceededReason &&
					applicationGroup.GetReadyCondition() == meta.WorkflowFailedReason &&
					applicationGroup.GetWorkflowCondition(v1alpha1.Rollback) == meta.SucceededReason
			}, DefaultTimeout, time.Second).Should(BeTrue())
		})

		It("should create the bookinfo spec and then delete it while in progress", func() {
			applicationGroup := defaultAppGroup(name)
			applicationGroup.Namespace = DefaultNamespace
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			By("Making sure that the workflow goes into a running state")
			Eventually(func() bool {
				workflow := &v1alpha12.Workflow{}
				workflowKey := types.NamespacedName{Name: applicationGroup.Name, Namespace: DefaultNamespace}
				_ = k8sClient.Get(ctx, workflowKey, workflow)
				return workflow.Status.Phase == v1alpha12.NodeRunning
			}, time.Minute, time.Second).Should(BeTrue())

			By("Waiting for the bookinfo object to reach a progressing reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.ProgressingReason
			}, time.Minute, time.Second).Should(BeTrue())

			By("Waiting for the ambassador helm release to be ready")
			Eventually(func() bool {
				helmRelease := &fluxhelmv2beta1.HelmRelease{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: ambassador, Namespace: name}, helmRelease); err != nil {
					return false
				}
				readyCondition := meta.GetResourceCondition(helmRelease, meta.ReadyCondition)
				if readyCondition == nil {
					return false
				}
				return readyCondition.Reason == meta2.ReconciliationSucceededReason
			}, time.Minute*2, time.Second).Should(BeTrue())

			// Wait for all the HelmReleases to delete
			err = k8sClient.Delete(ctx, applicationGroup)
			Expect(err).NotTo(HaveOccurred())

			By("Making sure that the workflow goes into a suspended state")
			Eventually(func() bool {
				workflow := &v1alpha12.Workflow{}
				workflowKey := types.NamespacedName{Name: applicationGroup.Name, Namespace: DefaultNamespace}
				_ = k8sClient.Get(ctx, workflowKey, workflow)
				return workflow.Spec.Suspend != nil && *workflow.Spec.Suspend
			}, time.Minute, time.Second).Should(BeTrue())

			By("waiting for the Workflow to delete all the HelmReleases")
			Eventually(func() bool {
				helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
				if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
					return false
				}
				return len(helmReleases.Items) == 0
			}, time.Minute*3, time.Second).Should(BeTrue())

			By("waiting for all the Workflows to be cleaned up from the cluster")
			Eventually(func() bool {
				workflowList := &v1alpha12.WorkflowList{}
				if err := k8sClient.List(ctx, workflowList, client.InNamespace(name)); err != nil {
					return false
				}
				return len(workflowList.Items) == 0
			}, time.Minute, time.Second).Should(BeTrue())
		})

		It("should delete the application group if reverse workflow is removed", func() {
			applicationGroup := smallAppGroup(name)
			applicationGroup.Name = name
			applicationGroup.Namespace = DefaultNamespace
			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			// Defer the cleanup so that we delete the appGroup after creation
			defer func() {
				By("Deleting the bookinfo object from the cluster")
				patch := client.MergeFrom(applicationGroup.DeepCopy())
				controllerutil.RemoveFinalizer(applicationGroup, v1alpha1.AppGroupFinalizer)
				_ = k8sClient.Patch(ctx, applicationGroup, patch)
				_ = k8sClient.Delete(ctx, applicationGroup)
			}()

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.SucceededReason
			}, DefaultTimeout, time.Second).Should(BeTrue())

			By("Deleting the application group and deleting the workflow")
			err = k8sClient.Delete(ctx, applicationGroup)
			Expect(err).To(BeNil())
			wf := &v1alpha12.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: DefaultNamespace,
				},
			}
			err = k8sClient.Delete(ctx, wf)
			Expect(err).To(BeNil())

			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				return errors.IsNotFound(k8sClient.Get(ctx, key, applicationGroup))
			}, time.Second*30, time.Second).Should(BeTrue())
		})
	})
})
