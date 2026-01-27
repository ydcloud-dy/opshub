{{/*
Expand the name of the chart.
*/}}
{{- define "opshub.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "opshub.fullname" -}}
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
{{- define "opshub.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "opshub.labels" -}}
helm.sh/chart: {{ include "opshub.chart" . }}
{{ include "opshub.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "opshub.selectorLabels" -}}
app.kubernetes.io/name: {{ include "opshub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Backend labels
*/}}
{{- define "opshub.backend.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{- define "opshub.backend.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Frontend labels
*/}}
{{- define "opshub.frontend.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: frontend
{{- end }}

{{- define "opshub.frontend.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: frontend
{{- end }}

{{/*
MySQL labels
*/}}
{{- define "opshub.mysql.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: mysql
{{- end }}

{{- define "opshub.mysql.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: mysql
{{- end }}

{{/*
Redis labels
*/}}
{{- define "opshub.redis.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{- define "opshub.redis.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{/*
MySQL host
*/}}
{{- define "opshub.mysql.host" -}}
{{- if .Values.mysql.enabled }}
{{- printf "%s-mysql" (include "opshub.fullname" .) }}
{{- else }}
{{- .Values.externalDatabase.host }}
{{- end }}
{{- end }}

{{/*
MySQL port
*/}}
{{- define "opshub.mysql.port" -}}
{{- if .Values.mysql.enabled }}
{{- printf "3306" }}
{{- else }}
{{- .Values.externalDatabase.port | toString }}
{{- end }}
{{- end }}

{{/*
MySQL database
*/}}
{{- define "opshub.mysql.database" -}}
{{- if .Values.mysql.enabled }}
{{- .Values.mysql.auth.database }}
{{- else }}
{{- .Values.externalDatabase.database }}
{{- end }}
{{- end }}

{{/*
Redis host
*/}}
{{- define "opshub.redis.host" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis" (include "opshub.fullname" .) }}
{{- else }}
{{- .Values.externalRedis.host }}
{{- end }}
{{- end }}

{{/*
Redis port
*/}}
{{- define "opshub.redis.port" -}}
{{- if .Values.redis.enabled }}
{{- printf "6379" }}
{{- else }}
{{- .Values.externalRedis.port | toString }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "opshub.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "opshub.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "opshub.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else }}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end }}
{{- end }}
