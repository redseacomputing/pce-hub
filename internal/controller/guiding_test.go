//go:build guiding

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pceredseacomputinggithubcomv1alpha1 "github.com/redseacomputing/pce-hub/api/v1alpha1"
)

// GUIDING TEST 4: Resilient Handling for Unreachable / Degraded Spokes
//
// Vertrag: Faellt ein Spoke aus, stuerzt der Hub-Operator nicht ab, sondern
// spiegelt den Zustand transparent: Condition SpokeAuditReady=False,
// Reason SpokeUnreachable, Phase=Degraded, geregelter Retry.
//
// SKIP-BEDINGUNG: Aktivierung, sobald Spoke-Meilenstein M1 erreicht ist
// ("Spoke veroeffentlicht Scans als OSCAL in die vom Hub gelesene Quelle").
var _ = Describe("Guiding Test 4: unreachable spoke sets degraded condition", func() {
	It("sets SpokeAuditReady=False with reason SpokeUnreachable instead of crashing", func() {

		report := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "guiding4-",
				Namespace:    "default",
			},
			Spec: pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReportSpec{
				// verweist auf eine Quelle, die der (ausgefallene) Spoke nicht beliefert
				ScanConfigMapRef: "spoke-source-with-no-data",
			},
		}
		Expect(k8sClient.Create(ctx, report)).To(Succeed())

		reconciler := &Nis2ComplianceReportReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: report.Name, Namespace: report.Namespace,
			},
		})

		// INVARIANTE 1: Reconcile schlaegt nicht fehl (kein Crash, kein Backoff-Sturm)
		Expect(err).NotTo(HaveOccurred())

		updated := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name: report.Name, Namespace: report.Namespace,
		}, updated)).To(Succeed())

		// INVARIANTE 2: transparenter Zustand statt Crash
		Expect(updated.Status.Phase).To(Equal("Degraded"))
		cond := meta.FindStatusCondition(updated.Status.Conditions, "SpokeAuditReady")
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal("SpokeUnreachable"))
	})
})
