# Copyright 2021 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
