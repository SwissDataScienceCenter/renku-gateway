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
{{- range $k, $v := .Values.global.core.versions -}}
{{- append $paths (printf "/api/renku/%s" $v.prefix) -}}
{{- if eq $k "latest" -}}
{{- append $paths "/api/renku" -}}
{{- end -}}
{{- end -}}
{{- join $paths "," | quote -}}
{{- end -}}
{{- end -}}

{{/*
Template core service names as a comma separated list
*/}}
{{- define "gateway.core.serviceNames" -}}
{{- $serviceNames := list -}}
{{- $coreBaseName := .Values.core.basename | default (printf "%s-core" .Release.Name -}}
{{- range $k, $v := .Values.global.core.versions -}}
{{- $serviceName := printf "%s-%s" $coreBaseName $v.name -}}
{{- append $serviceNames $serviceName -}}
{{- if eq $k "latest" -}}
{{- append $serviceNames $serviceName -}}
{{- end -}}
{{- end -}}
{{- join $serviceNames "," | quote -}}
{{- end -}}
