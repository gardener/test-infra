# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

{{- define "config" -}}
---
apiVersion: config.testmachinery.gardener.cloud/v1beta1
kind: BotConfiguration
{{ toYaml .Values.configuration }}

{{- end }}