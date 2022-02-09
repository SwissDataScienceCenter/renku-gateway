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

{{- define "redis.host" -}}
{{- if .Values.global.redis.host -}}
# If global hostname for redis is found use that
{{- .Values.global.redis.host -}}
{{- else -}}
{{- if hasKey .Subcharts "redis" -}}
# If global hostname for redis is not found, then check for redis subchart and then use that
{{- required "If a subchart for redis is used then redis.fullname should be defined." .Subcharts.redis.fullname -}}
{{- else -}}
# There is no redis subchart with fullname AND there is no hostname for redis in the global section - show error message
{{- required "Either global.redis.host should be defined or a subchart for redis should be used where redis.fullname is available." .Values.global.redis.host -}}
{{- end -}}
{{- end -}}
{{- end -}}
