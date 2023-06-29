{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "gateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "gateway.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "gateway.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Define URL protocol.
*/}}
{{- define "gateway.protocol" -}}
{{- if .Values.global.useHTTPS -}}
https
{{- else -}}
http
{{- end -}}
{{- end -}}

{{/*
Template core service paths as a comma separated list
*/}}
{{- define "gateway.core.paths" -}}
{{- $paths := list -}}
{{- range $i, $k := (keys .Values.global.core.versions | sortAlpha) -}}
{{- $paths = mustAppend $paths "/" -}}
{{- if eq $k "latest" -}}
{{- $paths = mustAppend $paths "/" -}}
{{- end -}}
{{- end -}}
{{- join "," $paths | quote -}}
{{- end -}}

{{/*
Template core service names as a comma separated list
*/}}
{{- define "gateway.core.serviceNames" -}}
{{- $serviceNames := list -}}
{{- $coreBaseName := .Values.core.basename | default (printf "%s-core" .Release.Name) -}}
{{- range $i, $k := (keys .Values.global.core.versions | sortAlpha) -}}
{{- $serviceName := printf "%s-%s" $coreBaseName (get $.Values.global.core.versions $k).name -}}
{{- $serviceNames = mustAppend $serviceNames $serviceName -}}
{{- if eq $k "latest" -}}
{{- $serviceNames = mustAppend $serviceNames $serviceName -}}
{{- end -}}
{{- end -}}
{{- join "," $serviceNames | quote -}}
{{- end -}}
