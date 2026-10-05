package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pceredseacomputinggithubcomv1alpha1 "github.com/redseacomputing/pce-hub/api/v1alpha1"
)

var _ = Describe("Nis2ComplianceReport Controller", func() {
	Context("When reconciling a resource", func() {
		It("foo", func() {
			ctx := context.Background()

			// Arrange: CRD in envtest anlegen
			res := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-report",
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(ctx, res)).To(Succeed())

			// Act: Reconciler manuell fuer die Ressource abrufen
			controller := &Nis2ComplianceReportReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-report",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				updatedResource := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-report",
					Namespace: "default",
				}, updatedResource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(updatedResource.Status.Conditions).NotTo(BeEmpty())
				var readyCond *metav1.Condition
				for _, c := range updatedResource.Status.Conditions {
					if c.Type == "Ready" {
						readyCond = &c
						break
					}
				}
				g.Expect(readyCond).NotTo(BeNil(), "Condition 'Ready' sollte existieren")
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(readyCond.Reason).To(Equal("ScanCompleted"))
			}, "5s", "100ms").Should(Succeed())
		})
	})
})
