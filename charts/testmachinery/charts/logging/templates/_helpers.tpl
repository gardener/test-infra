{{/*
Print a custom .Release.Name that can differ from a parent .Release.Name.
Defaults back to any inherited .Release.Name.
*/}}
{{- define "logging.releaseName" -}}
{{- default .Release.Name .Values.global.overwriteLoggingReleaseName -}}
{{- end -}}
