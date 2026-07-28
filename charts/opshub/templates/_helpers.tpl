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

{{- define "opshub.clickhouse.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: clickhouse
{{- end }}

{{- define "opshub.clickhouse.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: clickhouse
{{- end }}

{{/*
HA ClickHouse labels are intentionally distinct from standalone ClickHouse.
This prevents HA Services and PDBs from selecting the old Pod during migration.
*/}}
{{- define "opshub.clickhouse.haLabels" -}}
{{ include "opshub.clickhouse.labels" . }}
opshub.io/clickhouse-mode: ha
{{- end }}

{{- define "opshub.clickhouse.haSelectorLabels" -}}
{{ include "opshub.clickhouse.selectorLabels" . }}
opshub.io/clickhouse-mode: ha
{{- end }}

{{- define "opshub.clickhouse.endpoint" -}}
{{- if .Values.clickhouse.enabled -}}
{{- printf "http://%s-clickhouse:8123" (include "opshub.fullname" .) -}}
{{- else -}}
{{- .Values.externalClickHouse.endpoint -}}
{{- end -}}
{{- end }}

{{- define "opshub.clickhouse.database" -}}
{{- if .Values.clickhouse.enabled -}}
{{- .Values.clickhouse.auth.database -}}
{{- else -}}
{{- .Values.externalClickHouse.database -}}
{{- end -}}
{{- end }}

{{- define "opshub.clickhouse.username" -}}
{{- if .Values.clickhouse.enabled -}}
{{- .Values.clickhouse.auth.username -}}
{{- else -}}
{{- .Values.externalClickHouse.username -}}
{{- end -}}
{{- end }}

{{- define "opshub.clickhouse.host" -}}
{{- if .Values.clickhouse.enabled -}}
{{- printf "%s-clickhouse" (include "opshub.fullname" .) -}}
{{- else -}}
{{- $endpoint := .Values.externalClickHouse.endpoint | trimPrefix "http://" | trimPrefix "https://" -}}
{{- first (splitList ":" $endpoint) -}}
{{- end -}}
{{- end }}

{{- define "opshub.clickhouse.clusterName" -}}
{{- if and .Values.clickhouse.enabled .Values.clickhouse.highAvailability.enabled -}}
{{- .Values.clickhouse.highAvailability.clusterName -}}
{{- end -}}
{{- end }}

{{- define "opshub.logGateway.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: log-gateway
{{- end }}

{{- define "opshub.logGateway.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: log-gateway
{{- end }}

{{- define "opshub.logWriter.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: log-writer
{{- end }}

{{- define "opshub.logWriter.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: log-writer
{{- end }}

{{- define "opshub.redpanda.labels" -}}
{{ include "opshub.labels" . }}
app.kubernetes.io/component: redpanda
{{- end }}

{{- define "opshub.redpanda.selectorLabels" -}}
{{ include "opshub.selectorLabels" . }}
app.kubernetes.io/component: redpanda
{{- end }}

{{- define "opshub.kafka.brokers" -}}
{{- if .Values.logCenter.ingest.queue.brokers -}}
{{- join "," .Values.logCenter.ingest.queue.brokers -}}
{{- else if .Values.logCenter.ingest.queue.redpanda.enabled -}}
{{- printf "%s-redpanda:9092" (include "opshub.fullname" .) -}}
{{- else if eq (lower .Values.logCenter.ingest.queue.mode) "kafka" -}}
{{- fail "logCenter.ingest.queue.mode=kafka 时必须配置 queue.brokers 或启用 queue.redpanda" -}}
{{- end -}}
{{- end }}

{{- define "opshub.validate.logCenterHA" -}}
{{- $ingest := .Values.logCenter.ingest -}}
{{- if and $ingest.queue.redpanda.enabled (ne (lower $ingest.queue.mode) "kafka") -}}
{{- fail "启用内置 Redpanda 时 logCenter.ingest.queue.mode 必须为 kafka" -}}
{{- end -}}
{{- if $ingest.highAvailability.enabled -}}
{{- if ne (lower $ingest.queue.mode) "kafka" -}}
{{- fail "日志中心高可用模式必须使用 kafka 队列，不能使用 direct" -}}
{{- end -}}
{{- if lt (int $ingest.gateway.replicaCount) 2 -}}
{{- fail "日志中心高可用模式要求 Gateway 至少 2 个副本" -}}
{{- end -}}
{{- if lt (int $ingest.writer.replicaCount) 2 -}}
{{- fail "日志中心高可用模式要求 LogWriter 至少 2 个副本" -}}
{{- end -}}
{{- if lt (int $ingest.queue.partitions) (int $ingest.writer.replicaCount) -}}
{{- fail "日志中心高可用模式要求 Kafka 分区数不少于 LogWriter 副本数" -}}
{{- end -}}
{{- if $ingest.queue.redpanda.enabled -}}
{{- if lt (int $ingest.queue.redpanda.replicaCount) 3 -}}
{{- fail "日志中心高可用模式要求 Redpanda 至少 3 个 Broker" -}}
{{- end -}}
{{- if ne (lower $ingest.queue.redpanda.antiAffinity) "hard" -}}
{{- fail "日志中心高可用模式要求 Redpanda 使用 hard 反亲和" -}}
{{- end -}}
{{- if lt (int $ingest.queue.replicationFactor) 3 -}}
{{- fail "日志中心高可用模式要求 Kafka/Redpanda replicationFactor 至少为 3" -}}
{{- end -}}
{{- if gt (int $ingest.queue.replicationFactor) (int $ingest.queue.redpanda.replicaCount) -}}
{{- fail "Kafka replicationFactor 不能大于 Redpanda Broker 数" -}}
{{- end -}}
{{- if not $ingest.queue.redpanda.persistence.enabled -}}
{{- fail "日志中心高可用模式必须为 Redpanda 启用持久化" -}}
{{- end -}}
{{- end -}}
{{- if and .Values.clickhouse.enabled (not .Values.clickhouse.highAvailability.enabled) -}}
{{- fail "日志中心高可用模式使用内置 ClickHouse 时，必须启用 clickhouse.highAvailability" -}}
{{- end -}}
{{- if and .Values.clickhouse.enabled (ne (lower .Values.clickhouse.highAvailability.antiAffinity) "hard") -}}
{{- fail "日志中心高可用模式要求 ClickHouse 使用 hard 反亲和" -}}
{{- end -}}
{{- if and .Values.clickhouse.enabled (ne (lower .Values.clickhouse.highAvailability.keeper.antiAffinity) "hard") -}}
{{- fail "日志中心高可用模式要求 ClickHouse Keeper 使用 hard 反亲和" -}}
{{- end -}}
{{- if and (not .Values.clickhouse.enabled) (not .Values.externalClickHouse.endpoint) -}}
{{- fail "日志中心高可用模式必须启用内置 ClickHouse HA 或配置外部 ClickHouse 集群" -}}
{{- end -}}
{{- if and $ingest.writer.deadletter.persistence.enabled (not (has "ReadWriteMany" $ingest.writer.deadletter.persistence.accessModes)) -}}
{{- fail "LogWriter 多副本共享 deadletter PVC 时必须使用 ReadWriteMany" -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Frontend public URL for notification links.
Explicit server.frontendURL wins; otherwise derive it from the first Ingress host.
*/}}
{{- define "opshub.frontend.url" -}}
{{- if .Values.server.frontendURL }}
{{- .Values.server.frontendURL }}
{{- else if and .Values.ingress.enabled .Values.ingress.hosts }}
{{- $host := (index .Values.ingress.hosts 0).host -}}
{{- if $host }}
{{- $scheme := "http" -}}
{{- if .Values.ingress.tls }}
{{- $scheme = "https" -}}
{{- end }}
{{- printf "%s://%s" $scheme $host }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Server external URL for backend-facing links. It follows server.externalURL first,
then reuses the frontend URL derived above.
*/}}
{{- define "opshub.server.externalURL" -}}
{{- if .Values.server.externalURL }}
{{- .Values.server.externalURL }}
{{- else }}
{{- include "opshub.frontend.url" . }}
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
