{{/*
Expand the name of the chart.
*/}}
{{- define "artifusion.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "artifusion.fullname" -}}
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
{{- define "artifusion.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "artifusion.labels" -}}
helm.sh/chart: {{ include "artifusion.chart" . }}
{{ include "artifusion.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "artifusion.selectorLabels" -}}
app.kubernetes.io/name: {{ include "artifusion.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component labels for Artifusion
*/}}
{{- define "artifusion.artifusion.labels" -}}
{{ include "artifusion.labels" . }}
app.kubernetes.io/component: artifusion
{{- end }}

{{/*
Selector labels for Artifusion
*/}}
{{- define "artifusion.artifusion.selectorLabels" -}}
{{ include "artifusion.selectorLabels" . }}
app.kubernetes.io/component: artifusion
{{- end }}

{{/*
Component labels for OCI Registry
*/}}
{{- define "artifusion.ociRegistry.labels" -}}
{{ include "artifusion.labels" . }}
app.kubernetes.io/component: oci-registry
{{- end }}

{{/*
Selector labels for OCI Registry
*/}}
{{- define "artifusion.ociRegistry.selectorLabels" -}}
{{ include "artifusion.selectorLabels" . }}
app.kubernetes.io/component: oci-registry
{{- end }}

{{/*
Component labels for Registry
*/}}
{{- define "artifusion.registry.labels" -}}
{{ include "artifusion.labels" . }}
app.kubernetes.io/component: registry
{{- end }}

{{/*
Selector labels for Registry
*/}}
{{- define "artifusion.registry.selectorLabels" -}}
{{ include "artifusion.selectorLabels" . }}
app.kubernetes.io/component: registry
{{- end }}

{{/*
Component labels for Reposilite
*/}}
{{- define "artifusion.reposilite.labels" -}}
{{ include "artifusion.labels" . }}
app.kubernetes.io/component: reposilite
{{- end }}

{{/*
Selector labels for Reposilite
*/}}
{{- define "artifusion.reposilite.selectorLabels" -}}
{{ include "artifusion.selectorLabels" . }}
app.kubernetes.io/component: reposilite
{{- end }}

{{/*
Component labels for Verdaccio
*/}}
{{- define "artifusion.verdaccio.labels" -}}
{{ include "artifusion.labels" . }}
app.kubernetes.io/component: verdaccio
{{- end }}

{{/*
Selector labels for Verdaccio
*/}}
{{- define "artifusion.verdaccio.selectorLabels" -}}
{{ include "artifusion.selectorLabels" . }}
app.kubernetes.io/component: verdaccio
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "artifusion.serviceAccountName" -}}
{{- include "artifusion.fullname" . }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "artifusion.image" -}}
{{- $registryName := .Values.artifusion.image.repository -}}
{{- $tag := .Values.artifusion.image.tag | default .Chart.AppVersion -}}
{{- if .Values.global.imageRegistry }}
    {{- printf "%s/%s:%s" .Values.global.imageRegistry $registryName $tag -}}
{{- else }}
    {{- printf "%s:%s" $registryName $tag -}}
{{- end }}
{{- end }}

{{/*
Return the proper OCI Registry image name
*/}}
{{- define "artifusion.ociRegistry.image" -}}
{{- $registryName := .Values.ociRegistry.image.repository -}}
{{- $tag := .Values.ociRegistry.image.tag -}}
{{- if .Values.global.imageRegistry }}
    {{- printf "%s/%s:%s" .Values.global.imageRegistry $registryName $tag -}}
{{- else }}
    {{- printf "%s:%s" $registryName $tag -}}
{{- end }}
{{- end }}

{{/*
Return the proper Registry image name
*/}}
{{- define "artifusion.registry.image" -}}
{{- $registryName := .Values.registry.image.repository -}}
{{- $tag := .Values.registry.image.tag -}}
{{- if .Values.global.imageRegistry }}
    {{- printf "%s/%s:%s" .Values.global.imageRegistry $registryName $tag -}}
{{- else }}
    {{- printf "%s:%s" $registryName $tag -}}
{{- end }}
{{- end }}

{{/*
Return the proper Reposilite image name
*/}}
{{- define "artifusion.reposilite.image" -}}
{{- $registryName := .Values.reposilite.image.repository -}}
{{- $tag := .Values.reposilite.image.tag -}}
{{- if .Values.global.imageRegistry }}
    {{- printf "%s/%s:%s" .Values.global.imageRegistry $registryName $tag -}}
{{- else }}
    {{- printf "%s:%s" $registryName $tag -}}
{{- end }}
{{- end }}

{{/*
Return the proper Verdaccio image name
*/}}
{{- define "artifusion.verdaccio.image" -}}
{{- $registryName := .Values.verdaccio.image.repository -}}
{{- $tag := .Values.verdaccio.image.tag -}}
{{- if .Values.global.imageRegistry }}
    {{- printf "%s/%s:%s" .Values.global.imageRegistry $registryName $tag -}}
{{- else }}
    {{- printf "%s:%s" $registryName $tag -}}
{{- end }}
{{- end }}

{{/*
Return the storage class name to use
*/}}
{{- define "artifusion.storageClass" -}}
{{- if .Values.global.storageClass }}
    {{- .Values.global.storageClass -}}
{{- else if .storageClass }}
    {{- .storageClass -}}
{{- end }}
{{- end }}

{{/*
Service names
*/}}
{{- define "artifusion.artifusion.serviceName" -}}
{{- printf "%s-artifusion" (include "artifusion.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "artifusion.ociRegistry.serviceName" -}}
{{- printf "%s-oci-registry" (include "artifusion.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "artifusion.registry.serviceName" -}}
{{- printf "%s-registry" (include "artifusion.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "artifusion.reposilite.serviceName" -}}
{{- printf "%s-reposilite" (include "artifusion.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "artifusion.verdaccio.serviceName" -}}
{{- printf "%s-verdaccio" (include "artifusion.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Normalize imagePullSecrets to support both string and object formats
Usage: {{ include "artifusion.imagePullSecrets" . | nindent 8 }}
*/}}
{{- define "artifusion.imagePullSecrets" -}}
{{- range . }}
{{- if typeIs "string" . }}
- name: {{ . }}
{{- else }}
- {{ toYaml . | nindent 2 }}
{{- end }}
{{- end }}
{{- end }}
