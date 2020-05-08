# Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

{{- define "config" -}}
---
apiVersion: config.testmachinery.gardener.cloud/v1beta1
kind: Configuration
controller:
  healthAddr: ":{{ .Values.controller.healthEndpointPort }}"
  metricsAddr: ":{{ .Values.controller.metricsEndpointPort }}"
  enableLeaderElection: {{ .Values.controller.enableLeaderElection }}
  maxConcurrentSyncs: {{ .Values.controller.maxConcurrentSyncs }}
  webhook:
    port: {{ .Values.controller.webhook.port }}
    certDir: /etc/testmachinery-controller/srv

testmachinery:
  namespace: {{ .Release.Namespace }}
  testdefPath: {{ .Values.testmachinery.testdefPath }}
  local: {{ .Values.testmachinery.local }}
  insecure: {{ .Values.testmachinery.insecure }}
  disableCollector: {{ .Values.testmachinery.disableCollector }}
  cleanWorkflowPods: {{ .Values.testmachinery.cleanWorkflowPods }}

argo:
  argoUI:
    ingress:
      enabled: {{ .Values.testmachinery.argo.argoUI.ingress.enabled }}
      host: {{ .Values.testmachinery.argo.argoUI.ingress.host }}
{{- if .Values.testmachinery.argo.chartValues }}
  chartValues:
{{ toYaml .Values.testmachinery.argo.chartValues | indent 4 }}
{{- end }}

github:
  cache:
    cacheDir: {{ .Values.testmachinery.github.cache.cacheDir }}
    cacheDiskSizeGB: {{ .Values.testmachinery.github.cache.cacheDiskSizeGB }}
    maxAgeSeconds: {{ .Values.testmachinery.github.cache.maxAgeSeconds }}
  secretsPath: /etc/testmachinery-controller/secrets/git/github-secrets.yaml # mount secrets and specify the path

s3Configuration:
  server:
    {{- if .Values.testmachinery.s3Configuration.server.minio }}
    minio:
{{ toYaml .Values.testmachinery.s3Configuration.server.minio | indent 6 }}
    {{- end }}
    endpoint: {{ .Values.testmachinery.s3Configuration.server.endpoint }}
    ssl: {{ .Values.testmachinery.s3Configuration.server.ssl }}
  bucketName: {{ .Values.testmachinery.s3Configuration.bucketName }}
  accessKey: {{ .Values.testmachinery.s3Configuration.accessKey }}
  secretKey: {{ .Values.testmachinery.s3Configuration.secretKey }}

{{- if .Values.testmachinery.esConfiguration }}
esConfiguration:
{{ toYaml .Values.testmachinery.esConfiguration | indent 2 }}
{{- end }}

{{- if .Values.testmachinery.reservedExcessCapacity }}
reservedExcessCapacity:
{{ toYaml .Values.testmachinery.reservedExcessCapacity | indent 2 }}
{{- end }}

{{- if .Values.testmachinery.observability }}
observability:
{{ toYaml .Values.testmachinery.observability | indent 2 }}
{{- end }}

{{- end }}