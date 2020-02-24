package metadata

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SummaryType defines the type of a test result or summary
type SummaryType string

// Summary types can be testrun or teststep
const (
	SummaryTypeTestrun  SummaryType = "testrun"
	SummaryTypeTeststep SummaryType = "teststep"
)

// Metadata is the common metadata of all outputs and summaries.
type Metadata struct {
	// Short description of the flavor
	FlavorDescription string `json:"flavor_description,omitempty"`

	// Landscape describes the current dev,staging,canary,office or live.
	Landscape         string `json:"landscape,omitempty"`
	CloudProvider     string `json:"cloudprovider,omitempty"`
	KubernetesVersion string `json:"k8s_version,omitempty"`
	OperatingSystem   string `json:"operating_system,omitempty"`
	Region            string `json:"region,omitempty"`
	Zone              string `json:"zone,omitempty"`

	// ComponentDescriptor describes the current component_descriptor of the direct landscape-setup components.
	// It is formatted as an array of components: { name: "my_component", version: "0.0.1" }
	ComponentDescriptor interface{} `json:"bom,omitempty"`

	// Name of the testrun crd object.
	Testrun TestrunMetadata `json:"tr"`

	// all environment configuration values
	Configuration map[string]string `json:"config,omitempty"`

	// Additional annotations form the testrun or steps
	Annotations map[string]string `json:"annotations,omitempty"`

	// Represents how many retries the testrun had
	Retries int `json:"retries,omitempty"`

	// Contains the measured telemetry data
	// Is only used for internal sharing.
	TelemetryData *TelemetryData `json:"-"`
}

// TestrunMetadata represents the metadata of a testrun
type TestrunMetadata struct {
	// Name of the testrun crd object.
	ID string `json:"id"`

	// ID of the execution group this test belongs to
	ExecutionGroup string `json:"executionGroup,omitempty"`

	// StartTime of the testrun.
	StartTime *v1.Time `json:"startTime"`
}

// StepExportMetadata is the metadata of one step of a testrun.
type StepExportMetadata struct {
	Metadata
	StepName    string             `json:"stepName,omitempty"`
	TestDefName string             `json:"testdefinition,omitempty"`
	Phase       v1alpha1.NodePhase `json:"phase,omitempty"`
	StartTime   *v1.Time           `json:"startTime,omitempty"`
	Duration    int64              `json:"duration,omitempty"`
	PodName     string             `json:"podName"`
}

// TestrunSummary is the result of the overall testrun.
type TestrunSummary struct {
	Metadata      *Metadata          `json:"tm,omitempty"`
	Type          SummaryType        `json:"type,omitempty"`
	Phase         v1alpha1.NodePhase `json:"phase,omitempty"`
	StartTime     *v1.Time           `json:"startTime,omitempty"`
	Duration      int64              `json:"duration,omitempty"`
	TestsRun      int                `json:"testsRun,omitempty"`
	TelemetryData *TelemetryData     `json:"telemetry,omitempty"`
}

// StepSummary is the result of a specific step.
type StepSummary struct {
	Metadata    *Metadata          `json:"tm,omitempty"`
	Type        SummaryType        `json:"type,omitempty"`
	Name        string             `json:"name,omitempty"`
	StepName    string             `json:"stepName,omitempty"`
	Labels      []string           `json:"labels,omitempty"`
	Phase       v1alpha1.NodePhase `json:"phase,omitempty"`
	StartTime   *v1.Time           `json:"startTime,omitempty"`
	Duration    int64              `json:"duration,omitempty"`
	PreComputed *StepPreComputed   `json:"pre,omitempty"`
}

// StepPreComputed contains fields that could be created at runtime via scripted fields, but are created statically for better performance and better support of grafana
type StepPreComputed struct {
	// same as StepSummary.Phase but mapping states to ints (Failed&Timeout -> 0, Succeeded -> 100); allows to do averages on success rate in dashboards
	PhaseNum *int `json:"phaseNum,omitempty"`
	// A K8S Version without the patch suffix, e.g. "1.16"
	K8SMajorMinorVersion string `json:"k8sMajMinVer,omitempty"`
	// Dummy field for grafana/log links
	LogsDisplayName string `json:"logsText,omitempty"`
	// Dummy field for argoui/workflow links
	ArgoDisplayName string `json:"argoText,omitempty"`
	// the cluster domain of the testmachinery (useful to build other URLs in dashboards)
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

// Dimension describes the basic dimension of a test
type Dimension struct {
	Description       string `json:"description,omitempty"`
	Cloudprovider     string `json:"cloudprovider,omitempty"`
	KubernetesVersion string `json:"k8sVersion,omitempty"`
	OperatingSystem   string `json:"operating_system,omitempty"`
}

// TelemetryData describes the measured telemetry data for the tested shoot
type TelemetryData struct {
	ResponseTime    *TelemetryResponseTimeDuration `json:"response_time,omitempty"`
	DowntimePeriods *TelemetryDowntimePeriods      `json:"downtime,omitempty"`
}

// TelemetryResponseTimeDuration describes the response data of the telemetry measurement
type TelemetryResponseTimeDuration struct {
	Min    int   `json:"min"`
	Max    int   `json:"max"`
	Avg    int64 `json:"avg"`
	Median int64 `json:"median"`
	Std    int64 `json:"std"`
}

// TelemetryResponseTimeDuration describes the measured downtimes
type TelemetryDowntimePeriods struct {
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
	Avg    int64 `json:"avg"`
	Median int64 `json:"median"`
	Std    int64 `json:"std"`
}
