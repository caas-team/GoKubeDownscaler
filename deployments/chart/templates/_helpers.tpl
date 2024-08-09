{{/*
Expand the name of the chart.
*/}}
{{- define "go-kube-downscaler.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "go-kube-downscaler.fullname" -}}
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
{{- define "go-kube-downscaler.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "go-kube-downscaler.labels" -}}
application: {{ include "go-kube-downscaler.name" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "go-kube-downscaler.selectorLabels" -}}
application: {{ include "go-kube-downscaler.name" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "go-kube-downscaler.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "go-kube-downscaler.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
