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

// TestKubescapeHappyPath ist Guiding Test 2: echte Spoke-OSCAL-Daten in der
// Scan-Quelle muessen zu einem vollstaendig ausgewerteten Status fuehren.
func TestKubescapeHappyPath(t *testing.T) {
    ctx := context.Background()

    // Spoke-Realitaet: exakt das Format, das pkg/convert im pce-spoke erzeugt
    // (SC-8 = not-satisfied -> Art.21(2)h)
    oscalPayload := `{
      "assessment-results": {
        "uuid": "11111111-2222-3333-4444-555555555555",
        "metadata": {"title": "Kubescape Spoke Scan Result", "oscal-version": "1.1.2"},
        "results": [{
          "uuid": "66666666-7777-8888-9999-000000000000",
          "title": "Kubescape Spoke Scan",
          "findings": [{
            "target": {"target-id": "SC-8", "status": {"state": "not-satisfied"}},
            "description": "kubescape C-0035: Pod default/pod-with-hostpath"
          }]
        }]
      }
    }`

    ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "gt2-test"}}
    _ = k8sClient.Create(ctx, &ns)

    cm := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{Name: "gt2-scan-source", Namespace: ns.Name},
        Data:       map[string]string{"oscal": oscalPayload},
    }
    if err := k8sClient.Create(ctx, cm); err != nil {
        t.Fatalf("scan source setup failed: %v", err)
    }

    report := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{
        ObjectMeta: metav1.ObjectMeta{Name: "gt2-report", Namespace: ns.Name},
        Spec: pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReportSpec{
            ScanConfigMapRef: "gt2-scan-source",
        },
    }
    if err := k8sClient.Create(ctx, report); err != nil {
        t.Fatalf("report setup failed: %v", err)
    }

    reconciler := &Nis2ComplianceReportReconciler{
        Client: k8sClient, Scheme: k8sClient.Scheme(),
    }
    if _, err := reconciler.Reconcile(ctx, ctrl.Request{
        NamespacedName: types.NamespacedName{Namespace: ns.Name, Name: report.Name},
    }); err != nil {
        t.Fatalf("reconcile failed: %v", err)
    }

    // Assert: ausgewerteter Status
    updated := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{}
    if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(report), updated); err != nil {
        t.Fatalf("get updated report failed: %v", err)
    }

    if updated.Status.Phase != "Completed" {
        t.Errorf("expected Phase Completed, got %q", updated.Status.Phase)
    }
    cond := meta.FindStatusCondition(updated.Status.Conditions, "SpokeAuditReady")
    if cond == nil || cond.Status != metav1.ConditionTrue {
        t.Fatalf("expected SpokeAuditReady=True, got %+v", cond)
    }
    if updated.Status.OverallStatus != "Unsatisfied" {
        t.Errorf("expected OverallStatus Unsatisfied, got %q", updated.Status.OverallStatus)
    }
    var artH *pceredseacomputinggithubcomv1alpha1.ArticleComplianceSummary
    for i := range updated.Status.ArticleSummaries {
        if updated.Status.ArticleSummaries[i].Article == "Art.21(2)h" {
            artH = &updated.Status.ArticleSummaries[i]
        }
    }
    if artH == nil {
        t.Fatalf("expected summary for Art.21(2)h, got %+v", updated.Status.ArticleSummaries)
    }
    if artH.Status != "Unsatisfied" || artH.UnsatisfiedFindings < 1 {
        t.Errorf("expected Art.21(2)h Unsatisfied with >=1 unsatisfied finding, got %+v", artH)
    }
    if updated.Status.KubescapeScore == nil {
        t.Error("expected KubescapeScore to be set")
    }
    if updated.Status.LastEvaluated == nil {
        t.Error("expected LastEvaluated to be set")
    }
}
