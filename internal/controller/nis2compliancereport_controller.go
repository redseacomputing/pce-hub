/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pceredseacomputinggithubcomv1alpha1 "github.com/redseacomputing/pce-hub/api/v1alpha1"
	mapping "github.com/redseacomputing/pce/pkg/mapping"
  parser "github.com/redseacomputing/pce/pkg/parser"
)

// Nis2ComplianceReportReconciler reconciles a Nis2ComplianceReport object
type Nis2ComplianceReportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pce.redseacomputing.github.com,resources=nis2compliancereports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pce.redseacomputing.github.com,resources=nis2compliancereports/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pce.redseacomputing.github.com,resources=nis2compliancereports/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Nis2ComplianceReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Objekt laden
	var report pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport
	if err := r.Get(ctx, req.NamespacedName, &report); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Scan-Daten aus der referenzierten Quelle (Spoke) lesen
	  scanData, err := r.readScanData(ctx, &report)
    if err != nil {
        logger.Info("spoke unreachable / scan data missing", "error", err)
        return r.degradeSpoke(ctx, &report, err)
    }
    // 3. Guiding Test 2: OSCAL auswerten -> Scorecard -> Status
    if err := r.evaluateScan(ctx, &report, scanData); err != nil {
        logger.Error(err, "scan evaluation failed")
        report.Status.Phase = "Failed"
        _ = r.Status().Update(ctx, &report)
        return ctrl.Result{}, err
    }
    return ctrl.Result{}, nil
}
// degradeSpoke spiegelt einen nicht erreichbaren Spoke transparent im Status
// (Condition SpokeAuditReady=False, Reason SpokeUnreachable, Phase Degraded)
// und plant einen Retry - ohne Crash und ohne Backoff-Sturm.
func (r *Nis2ComplianceReportReconciler) degradeSpoke(ctx context.Context,
	report *pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport, cause error,
) (ctrl.Result, error) {
	meta.SetStatusCondition(&report.Status.Conditions, metav1.Condition{
		Type:               "SpokeAuditReady",
		Status:             metav1.ConditionFalse,
		Reason:             "SpokeUnreachable",
		Message:            fmt.Sprintf("spoke scan source not reachable: %v", cause),
		ObservedGeneration: report.Generation,
		LastTransitionTime: metav1.Now(),
	})
	report.Status.Phase = "Degraded"
	report.Status.ObservedGeneration = report.Generation

	if err := r.Status().Update(ctx, report); err != nil {
		return ctrl.Result{}, err // nur Status-Update-Fehler sind echte Fehler
	}
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// readScanData liest die OSCAL-Scan-Daten aus der in der Spec referenzierten ConfigMap.
func (r *Nis2ComplianceReportReconciler) readScanData(ctx context.Context,
	report *pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport,
) ([]byte, error) {
	key := "oscal"
	if report.Spec.ScanConfigMapKey != "" {
		key = report.Spec.ScanConfigMapKey
	}

	cmKey := types.NamespacedName{Namespace: report.Namespace, Name: report.Spec.ScanConfigMapRef}
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, cmKey, cm); err != nil {
		return nil, err
	}
	data, ok := cm.Data[key]
	if !ok {
		return nil, fmt.Errorf("configmap %s has no key %q", cmKey.Name, key)
	}
	return []byte(data), nil
}

// evaluateScan wertet die Spoke-OSCAL-Daten aus und schreibt den vollstaendigen
// Compliance-Status (Guiding Test 2). Normalisierung: Engine spricht klein-
// geschrieben ("unsatisfied"), die CRD-Enum verlangt "Unsatisfied".
func (r *Nis2ComplianceReportReconciler) evaluateScan(ctx context.Context,
    report *pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport, scanData []byte,
) error {
    results, err := parser.ParseAssessmentResults(bytes.NewReader(scanData))
    if err != nil {
        return fmt.Errorf("parse scan source: %w", err)
    }

    engine := mapping.NewPolicyCrosswalkEngine()
    if err := engine.LoadCatalogWithFallback(nil); err != nil {
        return fmt.Errorf("load mapping catalog: %w", err)
    }

    aggregates := engine.BuildScorecard(results[0])

    summaries := make([]pceredseacomputinggithubcomv1alpha1.ArticleComplianceSummary, 0, len(aggregates))
    overall, worst := "Unknown", 0 // 0=satisfied, 1=unknown, 2=unsatisfied
    satisfied, total := 0, 0
    for _, agg := range aggregates {
        s := pceredseacomputinggithubcomv1alpha1.ArticleComplianceSummary{
            Article:             string(agg.Article),
            Status:              normalizeStatus(string(agg.Status)),
            TotalFindings:       agg.TotalCount,
            UnsatisfiedFindings: agg.TotalCount - agg.SatisfiedCount,
        }
        summaries = append(summaries, s)
        satisfied += agg.SatisfiedCount
        total += agg.TotalCount
        switch agg.Status {
        case mapping.StatusUnsatisfied:
            worst = 2
        case mapping.StatusUnknown:
            if worst < 1 {
                worst = 1
            }
        }
    }
    switch worst {
    case 2:
        overall = "Unsatisfied"
    case 1:
        overall = "Unknown"
    case 0:
        overall = "Satisfied"
    }
    if len(aggregates) == 0 {
        overall = "Unknown" // keine Evidenz = unbekannt
    }

    var score int32 = 0
    if total > 0 {
        score = int32(100 * satisfied / total)
    }

    report.Status.Phase = "Completed"
    report.Status.OverallStatus = overall
    report.Status.ArticleSummaries = summaries
    report.Status.KubescapeScore = &score
    report.Status.LastEvaluated = &metav1.Time{Time: time.Now().UTC()}
    report.Status.ObservedGeneration = report.Generation
    meta.SetStatusCondition(&report.Status.Conditions, metav1.Condition{
        Type:               "SpokeAuditReady",
        Status:             metav1.ConditionTrue,
        Reason:             "SpokeAuditDelivered",
        Message:            "scan source evaluated",
        ObservedGeneration: report.Generation,
        LastTransitionTime: metav1.Now(),
    })
    return r.Status().Update(ctx, report)
}

// normalizeStatus hebt die Engine-Status-Sprache auf die CRD-Enum an.
func normalizeStatus(s string) string {
    switch s {
    case string(mapping.StatusSatisfied):
        return "Satisfied"
    case string(mapping.StatusUnsatisfied):
        return "Unsatisfied"
    case string(mapping.StatusNotApplicable):
        return "Unknown" // CRD-Enum hat kein NotApplicable (M2: Enum erweitern?)
    default:
        return "Unknown"
    }
}

// SetupWithManager sets up the controller with the Manager.
func (r *Nis2ComplianceReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pceredseacomputinggithubcomv1alpha1.Nis2ComplianceReport{}).
		Complete(r)
}
