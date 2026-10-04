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

// NIS2ComplianceReportSpec definiert den gewünschten Zustand des NIS2ComplianceReport.
type NIS2ComplianceReportSpec struct {
	// ScanConfigMapRef verweist auf die Quelle der OSCAL-Scan-Daten (z. B. eine ConfigMap von Kubescape/Falco/Velero).
	ScanConfigMapRef string `json:"scanConfigMapRef"`

	// CustomMappingConfigMapRef ist eine optionale benutzerdefinierte Mapping-Definition.
	// Wird diese nicht angegeben oder ist sie fehlerhaft, greift die Engine automatisch auf den einkompilierten Default-Katalog zurück.
	// +optional
	CustomMappingConfigMapRef *ConfigMapReference `json:"customMappingConfigMapRef,omitempty"`
}

// ArticleComplianceSummary fasst den Compliance-Status für einen einzelnen NIS2-Artikel zusammen.
type ArticleComplianceSummary struct {
	// Article ist der NIS2-Paragraph (z. B. "Art.21(2)f_Encryption")
	Article string `json:"article"`

	// Status beschreibt die Gesamtbewertung für diesen Artikel (Satisfied, Unsatisfied, Unknown)
	// +kubebuilder:validation:Enum=Satisfied;Unsatisfied;Unknown
	Status string `json:"status"`

	// TotalFindings ist die Anzahl aller zugeordneten Findings
	TotalFindings int `json:"totalFindings"`

	// UnsatisfiedFindings ist die Anzahl der fehlgeschlagenen Findings
	UnsatisfiedFindings int `json:"unsatisfiedFindings"`
}

// NIS2ComplianceReportStatus definiert den beobachteten Zustand nach der Engine-Ausführung.
type NIS2ComplianceReportStatus struct {
	// Phase beschreibt den aktuellen Zustand der Verarbeitung (Pending, Processing, Completed, Failed)
	// +kubebuilder:validation:Enum=Pending;Processing;Completed;Failed
	Phase string `json:"phase,omitempty"`

	// OverallStatus gibt den aggregierten Gesamtstatus an.
	// +kubebuilder:validation:Enum=Satisfied;Unsatisfied;Unknown
	OverallStatus string `json:"overallStatus,omitempty"`

	// ArticleSummaries enthält die feingranulare Zusammenfassung pro NIS2-Artikel.
	// +optional
	ArticleSummaries []ArticleComplianceSummary `json:"articleSummaries,omitempty"`

	// LastEvaluated zeigt den Zeitstempel der letzten Auswertung.
	// +optional
	LastEvaluated *metav1.Time `json:"lastEvaluated,omitempty"`

	// Conditions repräsentiert die Standard-Kubernetes-Conditions (z. B. Ready, Error)
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Overall Status",type="string",JSONPath=".status.overallStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// NIS2ComplianceReport ist das Schema für die Custom Resource der Policy Crosswalk Engine.
type NIS2ComplianceReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NIS2ComplianceReportSpec   `json:"spec,omitempty"`
	Status NIS2ComplianceReportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NIS2ComplianceReportList enthält eine Liste von NIS2ComplianceReport.
type NIS2ComplianceReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NIS2ComplianceReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NIS2ComplianceReport{}, &NIS2ComplianceReportList{})
}
