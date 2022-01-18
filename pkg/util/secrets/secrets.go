package secrets

import (
	"github.com/gardener/gardener/pkg/utils"
	"k8s.io/client-go/rest"

	render_template "github.com/gardener/test-infra/pkg/util/render-template"
)

// GenerateKubeconfigFromRestConfig generates a kubernetes kubeconfig from a rest client
func GenerateKubeconfigFromRestConfig(cfg *rest.Config, name string) ([]byte, error) {
	values := map[string]interface{}{
		"APIServerURL":      cfg.Host,
		"CACertificate":     utils.EncodeBase64(cfg.TLSClientConfig.CAData),
		"ClientCertificate": utils.EncodeBase64(cfg.TLSClientConfig.CertData),
		"ClientKey":         utils.EncodeBase64(cfg.TLSClientConfig.KeyData),
		"ClusterName":       name,
	}

	if cfg.Username != "" && cfg.Password != "" {
		values["BasicAuthUsername"] = cfg.Username
		values["BasicAuthPassword"] = cfg.Password
	}

	return render_template.RenderLocalTemplate(kubeconfigTemplate, values)
}

const kubeconfigTemplate = `---
apiVersion: v1
kind: Config
current-context: {{ .ClusterName }}
clusters:
- name: {{ .ClusterName }}
  cluster:
    certificate-authority-data: {{ .CACertificate }}
    server: https://{{ .APIServerURL }}
contexts:
- name: {{ .ClusterName }}
  context:
    cluster: {{ .ClusterName }}
{{- if and .ClientCertificate .ClientKey }}
    user: {{ .ClusterName }}
{{- else }}
    user: {{ .ClusterName }}-basic-auth
{{- end}}
users:
{{- if and .ClientCertificate .ClientKey }}
- name: {{ .ClusterName }}
  user:
    client-certificate-data: {{ .ClientCertificate }}
    client-key-data: {{ .ClientKey }}
{{- end}}
{{- if and .BasicAuthUsername .BasicAuthPassword }}
- name: {{ .ClusterName }}-basic-auth
  user:
    username: {{ .BasicAuthUsername }}
    password: {{ .BasicAuthPassword }}
{{- end}}`
