{{/*
AI Control Plane - Helm Chart Helpers
Common naming and labeling functions
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "acp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "acp.fullname" -}}
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
{{- define "acp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "acp.labels" -}}
helm.sh/chart: {{ include "acp.chart" . }}
{{ include "acp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- range $key, $value := .Values.commonLabels }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "acp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "acp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
LiteLLM component labels
*/}}
{{- define "acp.litellmLabels" -}}
{{ include "acp.labels" . }}
app.kubernetes.io/component: litellm
{{- end }}

{{/*
LiteLLM selector labels
*/}}
{{- define "acp.litellmSelectorLabels" -}}
{{ include "acp.selectorLabels" . }}
app.kubernetes.io/component: litellm
{{- end }}

{{/*
PostgreSQL component labels
*/}}
{{- define "acp.postgresLabels" -}}
{{ include "acp.labels" . }}
app.kubernetes.io/component: postgres
{{- end }}

{{/*
PostgreSQL selector labels
*/}}
{{- define "acp.postgresSelectorLabels" -}}
{{ include "acp.selectorLabels" . }}
app.kubernetes.io/component: postgres
{{- end }}

{{/*
Mock Upstream component labels
*/}}
{{- define "acp.mockUpstreamLabels" -}}
{{ include "acp.labels" . }}
app.kubernetes.io/component: mock-upstream
{{- end }}

{{/*
Mock Upstream selector labels
*/}}
{{- define "acp.mockUpstreamSelectorLabels" -}}
{{ include "acp.selectorLabels" . }}
app.kubernetes.io/component: mock-upstream
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "acp.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "acp.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
LiteLLM secret name
*/}}
{{- define "acp.litellmSecretName" -}}
{{- if .Values.secrets.create }}
{{- printf "%s-litellm" (include "acp.fullname" .) }}
{{- else }}
{{- required "secrets.existingSecret.name is required when secrets.create is false" .Values.secrets.existingSecret.name }}
{{- end }}
{{- end }}

{{/*
Database secret name
*/}}
{{- define "acp.dbSecretName" -}}
{{- if .Values.postgres.enabled }}
{{- printf "%s-postgres" (include "acp.fullname" .) }}
{{- else if .Values.secrets.create }}
{{- printf "%s-litellm" (include "acp.fullname" .) }}
{{- else }}
{{- required "externalDatabase.existingSecret or secrets.existingSecret.name is required when postgres.enabled is false" (default .Values.externalDatabase.existingSecret .Values.secrets.existingSecret.name) }}
{{- end }}
{{- end }}

{{/*
Database secret key
*/}}
{{- define "acp.dbSecretKey" -}}
{{- if .Values.postgres.enabled -}}
DATABASE_URL
{{- else if .Values.externalDatabase.existingSecret -}}
{{ .Values.externalDatabase.existingSecretKey }}
{{- else -}}
{{ .Values.secrets.existingSecret.databaseUrlKey }}
{{- end -}}
{{- end }}

{{/*
PostgreSQL service name
*/}}
{{- define "acp.postgresService" -}}
{{- printf "%s-postgres" (include "acp.fullname" .) }}
{{- end }}

{{/*
PostgreSQL full service name (FQDN)
*/}}
{{- define "acp.postgresServiceFQDN" -}}
{{- printf "%s.%s.svc.cluster.local" (include "acp.postgresService" .) .Release.Namespace }}
{{- end }}

{{/*
Mock Upstream service name
*/}}
{{- define "acp.mockUpstreamService" -}}
{{- printf "%s-mock-upstream" (include "acp.fullname" .) }}
{{- end }}

{{/*
LiteLLM config name
*/}}
{{- define "acp.litellmConfigName" -}}
{{- printf "%s-litellm-config" (include "acp.fullname" .) }}
{{- end }}

{{/*
Generate database URL for embedded PostgreSQL
*/}}
{{- define "acp.embeddedDatabaseUrl" -}}
{{- printf "postgresql://%s:%s@%s:5432/%s" .Values.postgres.auth.username .Values.postgres.auth.password (include "acp.postgresServiceFQDN" .) .Values.postgres.auth.database }}
{{- end }}
