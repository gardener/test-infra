package strconf

import (
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
)

// ConfigSource represents a source for the value of a config element.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ConfigSource struct {
	// Selects a key of a ConfigMap.
	// +optional
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret in the pod's namespace
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// StringOrConfig represents a type that could be from a string or a configuration
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type StringOrConfig struct {
	Type      Type
	StrVal    string
	ConfigVal ConfigSource
}

// Type represents the stored type of IntOrString.
type Type int

const (
	String Type = iota // The IntOrString holds an int.
	Config             // The IntOrString holds a string.
)

// FromString creates a StringOrConfig from a string.
func FromString(s string) *StringOrConfig {
	return &StringOrConfig{
		Type:   String,
		StrVal: s,
	}
}

// FromConfig creates a StringOrConfoig from a ConfigSoource.
func FromConfig(c ConfigSource) *StringOrConfig {
	return &StringOrConfig{
		Type:      Config,
		ConfigVal: c,
	}
}

// String returns the strconf as a string value
func (strsec *StringOrConfig) String() string {
	return strsec.StrVal
}

// Config returns the strconf as a Config struct
func (strsec *StringOrConfig) Config() *ConfigSource {
	return &strsec.ConfigVal
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (strsec *StringOrConfig) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		strsec.Type = String
		return json.Unmarshal(value, &strsec.StrVal)
	}
	strsec.Type = Config
	return json.Unmarshal(value, &strsec.ConfigVal)
}

// MarshalJSON implements the json.Marshaller interface.
func (strsec *StringOrConfig) MarshalJSON() ([]byte, error) {
	switch strsec.Type {
	case String:
		return json.Marshal(strsec.StrVal)
	case Config:
		return json.Marshal(strsec.ConfigVal)
	default:
		return []byte{}, fmt.Errorf("impossible StringOrConfig.Type")
	}
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ StringOrConfig) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ StringOrConfig) OpenAPISchemaFormat() string { return "string-or-secretref" }
