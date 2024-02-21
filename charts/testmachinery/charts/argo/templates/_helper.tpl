# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

{{- define "getDefaultLoggingUrl" -}}
{{- $grafanaHost := "" }}
{{- if .Values.argo.logging.grafana.https  -}}
{{- $grafanaHost = printf "https://%s" .Values.argo.logging.grafana.host -}}
{{- else -}}
{{- $grafanaHost = printf "http://%s" .Values.argo.logging.grafana.host -}}
{{- end -}}
{{- $pathWorkflow := "/explore?left=[\"now-3d\",\"now\",\"Loki\",{\"expr\":\"{container%3D\\\"main\\\",argo_workflow%3D\\\"${metadata.name}\\\"}\"},{\"mode\":\"Logs\"},{\"ui\":[true,true,true,\"exact\"]}]" -}}
{{- $pathPod := "/explore?left=[\"now-3d\",\"now\",\"Loki\",{\"expr\":\"{container%3D\\\"main\\\",instance%3D\\\"${metadata.name}\\\"}\"},{\"mode\":\"Logs\"},{\"ui\":[true,true,true,\"exact\"]}]" -}}
- name: "Grafana Pod Log"
  scope: "pod"
  url:  {{ printf "%s%s" $grafanaHost $pathPod }}
- name: "Grafana Workflow Log"
  scope: "workflow"
  url: {{ printf "%s%s" $grafanaHost $pathWorkflow }}
{{- end -}}
