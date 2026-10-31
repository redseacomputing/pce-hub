package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	corev1 "k8s.io/api/core/v1"

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

	    Context("when the spoke delivers real OSCAL scan data", func() {
          It("evaluates the scan and reports completed compliance status", func() {
                ctx := context.Background()

                // Spoke-Realitaet: exakt das Format, das pkg/convert im
                // pce-spoke erzeugt (SC-8 = not-satisfied -> Art.21(2)h)
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

                // Arrange: Scan-Quelle (vom Spoke beschrieben) + verweisender Report
                source := &corev1.ConfigMap{
                    ObjectMeta: metav1.ObjectMeta{Name: "gt2-scan-source", Namespace: "default"},
                    Data:       map[string]string{"oscal": oscalPayload},
                }
                Expect(k8sClient.Create(ctx, source)).To(Succeed())

                report := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{
                    ObjectMeta: metav1.ObjectMeta{Name: "gt2-report", Namespace: "default"},
                    Spec: pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReportSpec{
                        ScanConfigMapRef: "gt2-scan-source",
                    },
                }
                Expect(k8sClient.Create(ctx, report)).To(Succeed())

                // Act
                controller := &Nis2ComplianceReportReconciler{
                    Client: k8sClient,
                    Scheme: k8sClient.Scheme(),
                }
                _, err := controller.Reconcile(ctx, reconcile.Request{
                    NamespacedName: types.NamespacedName{Name: "gt2-report", Namespace: "default"},
                })
                Expect(err).NotTo(HaveOccurred())

                // Assert: vollstaendig ausgewerteter Status (Guiding Test 2)
                updated := &pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{}
                Expect(k8sClient.Get(ctx, types.NamespacedName{
                    Name: "gt2-report", Namespace: "default",
                }, updated)).To(Succeed())

                Expect(updated.Status.Phase).To(Equal("Completed"))

                cond := meta.FindStatusCondition(updated.Status.Conditions, "SpokeAuditReady")
                Expect(cond).NotTo(BeNil(), "Condition 'SpokeAuditReady' sollte existieren")
                Expect(cond.Status).To(Equal(metav1.ConditionTrue))

                Expect(updated.Status.OverallStatus).To(Equal("Unsatisfied"))

                var artH *pceredseacomputinggithubcomv1alpha1.ArticleComplianceSummary
                for i := range updated.Status.ArticleSummaries {
                    if updated.Status.ArticleSummaries[i].Article == "Art.21(2)h" {
                        artH = &updated.Status.ArticleSummaries[i]
                    }
                }
                Expect(artH).NotTo(BeNil(), "Art.21(2)h sollte in den Summaries sein")
                Expect(artH.Status).To(Equal("Unsatisfied"))
                Expect(artH.UnsatisfiedFindings).To(BeNumerically(">=", 1))

                Expect(updated.Status.KubescapeScore).NotTo(BeNil(), "KubescapeScore sollte gesetzt sein")
                Expect(updated.Status.LastEvaluated).NotTo(BeNil(), "LastEvaluated sollte gesetzt sein")
          })
      })

})
