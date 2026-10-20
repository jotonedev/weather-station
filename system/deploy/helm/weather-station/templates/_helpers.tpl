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
Component fullname helper: (include "weather-station.componentFullname" (dict "root" . "component" "database"))
*/}}
{{- define "weather-station.componentFullname" -}}
{{- printf "%s-%s" (include "weather-station.fullname" .root) .component | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Component labels: (include "weather-station.componentLabels" (dict "root" . "component" "database"))
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
Component selector labels: (include "weather-station.componentSelectorLabels" (dict "root" . "component" "database"))
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
# Track only the digest of the tag in use (e.g. latest). Without this, the
# force policy's multi-tag watcher treats any other tag (incl. SHA tags) as
# a valid update candidate and can pin stale/broken builds.
keel.sh/match-tag: "true"
{{- end }}
{{- end }}

{{/*
Deployment annotations block (Keel + user annotations) for a component.
Usage:
  {{- $annotations := include "weather-station.componentAnnotations" (dict "root" . "component" .Values.<component>) | trim }}
  {{- if $annotations }}{{- $annotations | nindent 2 }}{{- end }}
*/}}
{{- define "weather-station.componentAnnotations" -}}
{{- if or .root.Values.keel.enabled .component.annotations }}
annotations:
  {{- include "weather-station.keelAnnotations" .root | nindent 2 }}
  {{- with .component.annotations }}
  {{- toYaml . | nindent 2 }}
  {{- end }}
{{- end }}
{{- end }}

{{/*
Placement settings (nodeSelector / affinity / tolerations) for a component.
Usage:
  {{- $placement := include "weather-station.placement" (dict "component" .Values.<component>) | trim }}
  {{- if $placement }}{{- $placement | nindent 6 }}{{- end }}
*/}}
{{- define "weather-station.placement" -}}
{{- with .component.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .component.affinity }}
affinity:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .component.tolerations }}
tolerations:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}

