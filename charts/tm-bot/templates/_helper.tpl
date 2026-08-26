# SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
#
# SPDX-License-Identifier: Apache-2.0

{{- define "config" -}}
---
apiVersion: config.testmachinery.gardener.cloud/v1beta1
kind: BotConfiguration
{{ toYaml .Values.configuration }}

{{- end }}