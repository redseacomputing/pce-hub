package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pceredseacomputinggithubcomv1alpha1 "github.com/redseacomputing/pce-hub/api/v1alpha1"
)

var _ = Describe("Nis2ComplianceReport Controller", func() {
	Context("When reconciling a resource", func() {
		It("degrades gracefully when the scan source configmap does not exist", func() {
			ctx := context.Background()

			// Arrange: CRD in envtest anlegen; ScanConfigMapRef zeigt auf
			// eine NICHT existierende ConfigMap (= ausgefallener Spoke)
			res := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-report",
					Namespace: "default",
				},
				Spec: pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReportSpec{
					ScanConfigMapRef: "test-source",
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

			// Assert 1: Reconcile selbst schlaegt nicht fehl
			Expect(err).NotTo(HaveOccurred())

			// Assert 2: transparenter Degraded-Zustand statt Crash
			updated := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-report",
				Namespace: "default",
			}, updated)).To(Succeed())

			Expect(updated.Status.Phase).To(Equal("Degraded"))

			cond := meta.FindStatusCondition(updated.Status.Conditions, "SpokeAuditReady")
			Expect(cond).NotTo(BeNil(), "Condition 'SpokeAuditReady' sollte existieren")
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("SpokeUnreachable"))
		})
	})
})
