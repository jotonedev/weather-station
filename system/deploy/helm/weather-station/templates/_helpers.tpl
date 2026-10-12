{{/*
Expand the name of the chart.
*/}}
{{- define "weather-station.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "weather-station.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "weather-station.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "weather-station.labels" -}}
helm.sh/chart: {{ include "weather-station.chart" . }}
{{ include "weather-station.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "weather-station.selectorLabels" -}}
app.kubernetes.io/name: {{ include "weather-station.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component fullname helper: (include "weather-station.componentFullname" (dict "root" . "component" "questdb"))
*/}}
{{- define "weather-station.componentFullname" -}}
{{- printf "%s-%s" (include "weather-station.fullname" .root) .component | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Component labels: (include "weather-station.componentLabels" (dict "root" . "component" "questdb"))
*/}}
{{- define "weather-station.componentLabels" -}}
helm.sh/chart: {{ include "weather-station.chart" .root }}
{{ include "weather-station.componentSelectorLabels" (dict "root" .root "component" .component) }}
{{- if .root.Chart.AppVersion }}
app.kubernetes.io/version: {{ .root.Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
{{- end }}

{{/*
Component selector labels: (include "weather-station.componentSelectorLabels" (dict "root" . "component" "questdb"))
*/}}
{{- define "weather-station.componentSelectorLabels" -}}
app.kubernetes.io/name: {{ include "weather-station.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Keel annotations helper
*/}}
{{- define "weather-station.keelAnnotations" -}}
{{- if .Values.keel.enabled }}
keel.sh/policy: {{ .Values.keel.policy | quote }}
keel.sh/trigger: {{ .Values.keel.trigger | quote }}
keel.sh/pollSchedule: {{ .Values.keel.pollSchedule | quote }}
{{- end }}
{{- end }}

