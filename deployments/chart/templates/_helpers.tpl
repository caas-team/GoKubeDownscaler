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
If replicaCount is greater than 1 leader election is enabled by default.
*/}}
{{- define "go-kube-downscaler.leaderElection" -}}
{{- if gt (.Values.replicaCount | int) 1 }}true{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "go-kube-downscaler.labels" -}}
helm.sh/chart: {{ include "go-kube-downscaler.chart" . }}
{{ include "go-kube-downscaler.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "go-kube-downscaler.selectorLabels" -}}
application: {{ include "go-kube-downscaler.fullname" . }}
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

{{/*
Create defined permissions for roles
*/}}
{{- define "go-kube-downscaler.permissions" -}}
- apiGroups:
    - ""
  resources:
    - namespaces
  verbs:
    - get
- apiGroups:
    - ""
  resources:
    - events
  verbs:
    - get
    - create
    - update
{{- range $resource := .Values.includedResources }}
{{- if eq $resource "deployments" }}
- apiGroups:
    - apps
  resources:
    - deployments
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "statefulsets" }}
- apiGroups:
    - apps
  resources:
    - statefulsets
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "daemonsets" }}
- apiGroups:
    - apps
  resources:
    - daemonsets
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "rollouts" }}
- apiGroups:
    - argoproj.io
  resources:
    - rollouts
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "horizontalpodautoscalers" }}
- apiGroups:
    - autoscaling
  resources:
    - horizontalpodautoscalers
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "jobs" }}
- apiGroups:
    - batch
  resources:
    - jobs
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "cronjobs" }}
- apiGroups:
    - batch
  resources:
    - cronjobs
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "scaledobjects" }}
- apiGroups:
    - keda.sh
  resources:
    - scaledobjects
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "stacks" }}
- apiGroups:
    - zalando.org
  resources:
    - stacks
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "prometheuses" }}
- apiGroups:
    - monitoring.coreos.com
  resources:
    - prometheuses
  verbs:
    - get
    - list
    - update
{{- end }}
{{- if eq $resource "poddisruptionbudgets" }}
- apiGroups:
    - policy
  resources:
    - poddisruptionbudgets
  verbs:
    - get
    - list
    - update
{{- end }}
{{- end }}
{{- end }}
