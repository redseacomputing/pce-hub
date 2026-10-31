package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapReference verweist auf eine benutzerspezifische Mapping-ConfigMap im selben Namespace.
type ConfigMapReference struct {
	// Name der ConfigMap
	Name string `json:"name"`
	// Key innerhalb der ConfigMap, der die YAML-Definition enthält (Default: "mapping.yaml")
	// +optional
	Key string `json:"key,omitempty"`
}

// Nis2ComplianceReportSpec definiert den gewünschten Zustand des Nis2ComplianceReport.
type Nis2ComplianceReportSpec struct {
	// ScanConfigMapRef verweist auf die Quelle der OSCAL-Scan-Daten (z. B. eine ConfigMap von Kubescape/Falco/Velero).
	ScanConfigMapRef string `json:"scanConfigMapRef"`

	// ScanConfigMapKey ist der Key innerhalb der Scan-ConfigMap (Default: "oscal").
	// +optional
	ScanConfigMapKey string `json:"scanConfigMapKey,omitempty"`

	// CustomMappingConfigMapRef ist eine optionale benutzerdefinierte Mapping-Definition.
	// Wird diese nicht angegeben oder ist sie fehlerhaft, greift die Engine automatisch auf den einkompilierten Default-Katalog zurück.
	// +optional
	CustomMappingConfigMapRef *ConfigMapReference `json:"customMappingConfigMapRef,omitempty"`

	// ThreatConfigMapRef verweist optional auf die Falco-Threat-Quelle des Spokes
  // (aggregierte Laufzeit-Anomalien). Fehlt sie, bleibt ActiveThreatCount
  // undefiniert (keine Evidenz) - das ist KEINE Degradation.
  // +optional
  ThreatConfigMapRef string `json:"threatConfigMapRef,omitempty"`

  // ThreatConfigMapKey ist der Key innerhalb der Threat-ConfigMap (Default: "threat").
  // +optional
  ThreatConfigMapKey string `json:"threatConfigMapKey,omitempty"`
}

// ArticleComplianceSummary fasst den Compliance-Status für einen einzelnen NIS2-Artikel zusammen.
type ArticleComplianceSummary struct {
	// Article ist der NIS2-Paragraph (z. B. "Art.21(2)f_Encryption")
	Article string `json:"article"`

	// Status beschreibt die Gesamtbewertung für diesen Artikel (Satisfied, Unsatisfied, Unknown)
	// +kubebuilder:validation:Enum=Satisfied;Unsatisfied;Unknown;NotApplicable
	Status string `json:"status"`

	// TotalFindings ist die Anzahl aller zugeordneten Findings
	TotalFindings int `json:"totalFindings"`

	// UnsatisfiedFindings ist die Anzahl der fehlgeschlagenen Findings
	UnsatisfiedFindings int `json:"unsatisfiedFindings"`
}

// Nis2ComplianceReportStatus definiert den beobachteten Zustand nach der Engine-Ausführung.
type Nis2ComplianceReportStatus struct {
	// Phase beschreibt den aktuellen Zustand der Verarbeitung (Pending, Processing, Completed, Degraded, Failed).
	// Degraded = Spoke nicht erreichbar / Scan-Daten unvollständig, Retry läuft.
	// +kubebuilder:validation:Enum=Pending;Processing;Completed;Degraded;Failed
	Phase string `json:"phase,omitempty"`

	// OverallStatus gibt den aggregierten Gesamtstatus an.
	// +kubebuilder:validation:Enum=Satisfied;Unsatisfied;Unknown
	OverallStatus string `json:"overallStatus,omitempty"`

	// ArticleSummaries enthält die feingranulare Zusammenfassung pro NIS2-Artikel.
	// +optional
	ArticleSummaries []ArticleComplianceSummary `json:"articleSummaries,omitempty"`

	// KubescapeScore ist der aggregierte Benchmark-Score des letzten Spoke-Audits (0-100).
	// +optional
	KubescapeScore *int32 `json:"kubescapeScore,omitempty"`

	// ActiveThreatCount spiegelt aktive Laufzeit-Anomalien (Falco), ohne Remediation.
	// +optional
	ActiveThreatCount *int32 `json:"activeThreatCount,omitempty"`

	// LastEvaluated zeigt den Zeitstempel der letzten Auswertung.
	// +optional
	LastEvaluated *metav1.Time `json:"lastEvaluated,omitempty"`

	// ObservedGeneration ist die Generation, deren Spec zuletzt ausgewertet wurde.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions repräsentiert die Standard-Kubernetes-Conditions (z. B. SpokeAuditReady, Ready).
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Overall Status",type="string",JSONPath=".status.overallStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Nis2ComplianceReport ist das Schema für die Custom Resource der Policy Crosswalk Engine.
type Nis2ComplianceReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Nis2ComplianceReportSpec   `json:"spec,omitempty"`
	Status Nis2ComplianceReportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Nis2ComplianceReportList enthält eine Liste von Nis2ComplianceReport.
type Nis2ComplianceReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nis2ComplianceReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nis2ComplianceReport{}, &Nis2ComplianceReportList{})
}
