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

{{/*
Validate secure-by-default production contract.
*/}}
{{- define "acp.validate" -}}
{{- $root := . -}}
{{- $demoEnabled := default false .Values.demo.enabled -}}
{{- $ingressTls := default (list) .Values.ingress.tls -}}
{{- $ingressHosts := default (list) .Values.ingress.hosts -}}
{{- $publicUrl := default "" .Values.litellm.publicUrl -}}
{{- if and $demoEnabled (ne .Values.profile "demo") -}}
{{- fail "demo.enabled=true requires profile=demo." -}}
{{- end -}}
{{- if and (eq .Values.profile "demo") (not $demoEnabled) -}}
{{- fail "profile=demo requires demo.enabled=true and an explicitly named demo-only values file." -}}
{{- end -}}
{{- if not $demoEnabled -}}
  {{- if ne .Values.profile "production" -}}
  {{- fail "Primary Helm path is production-only. Use examples/values.demo.yaml or examples/values.offline.yaml for demo deployments." -}}
  {{- end -}}
  {{- if .Values.secrets.create -}}
  {{- fail "secrets.create must be false for the production profile. Provide externally managed secrets instead." -}}
  {{- end -}}
  {{- if not .Values.secrets.existingSecret.name -}}
  {{- fail "secrets.existingSecret.name is required for the production profile." -}}
  {{- end -}}
  {{- if .Values.postgres.enabled -}}
  {{- fail "postgres.enabled must be false for the production profile. Use an external PostgreSQL service." -}}
  {{- end -}}
  {{- if not (or .Values.externalDatabase.existingSecret .Values.secrets.existingSecret.name) -}}
  {{- fail "externalDatabase.existingSecret or secrets.existingSecret.name must supply DATABASE_URL for the production profile." -}}
  {{- end -}}
  {{- if ne .Values.litellm.service.type "ClusterIP" -}}
  {{- fail "litellm.service.type must remain ClusterIP for the production profile." -}}
  {{- end -}}
  {{- if ne .Values.litellm.mode "online" -}}
  {{- fail "litellm.mode must be online for the production profile. Demo/offline behavior belongs in demo-only examples." -}}
  {{- end -}}
  {{- if not .Values.networkPolicy.enabled -}}
  {{- fail "networkPolicy.enabled must be true for the production profile." -}}
  {{- end -}}
  {{- if not .Values.podDisruptionBudget.enabled -}}
  {{- fail "podDisruptionBudget.enabled must be true for the production profile." -}}
  {{- end -}}
  {{- if .Values.autoscaling.enabled -}}
    {{- if lt (int .Values.autoscaling.minReplicas) 2 -}}
    {{- fail "autoscaling.minReplicas must be at least 2 for the production profile." -}}
    {{- end -}}
  {{- else if lt (int .Values.litellm.replicaCount) 2 -}}
  {{- fail "litellm.replicaCount must be at least 2 when autoscaling is disabled in the production profile." -}}
  {{- end -}}
  {{- if not .Values.litellm.podSecurityContext.runAsNonRoot -}}
  {{- fail "litellm.podSecurityContext.runAsNonRoot must be true for the production profile." -}}
  {{- end -}}
  {{- if not .Values.litellm.securityContext.runAsNonRoot -}}
  {{- fail "litellm.securityContext.runAsNonRoot must be true for the production profile." -}}
  {{- end -}}
  {{- if .Values.monitoring.serviceMonitor.tlsConfig.insecureSkipVerify -}}
  {{- fail "monitoring.serviceMonitor.tlsConfig.insecureSkipVerify must remain false for the production profile." -}}
  {{- end -}}
  {{- if and $publicUrl (not (hasPrefix "https://" $publicUrl)) -}}
  {{- fail "litellm.publicUrl must start with https:// when set." -}}
  {{- end -}}
  {{- if .Values.ingress.enabled -}}
    {{- $hasAlbCertificate := and (kindIs "map" .Values.ingress.annotations) (hasKey .Values.ingress.annotations "alb.ingress.kubernetes.io/certificate-arn") -}}
    {{- if not .Values.ingress.className -}}
    {{- fail "ingress.className is required when ingress.enabled=true." -}}
    {{- end -}}
    {{- if eq (len $ingressHosts) 0 -}}
    {{- fail "ingress.hosts must contain at least one hostname when ingress.enabled=true." -}}
    {{- end -}}
    {{- if and (eq (len $ingressTls) 0) (not $hasAlbCertificate) -}}
    {{- fail "ingress.enabled=true requires either ingress.tls entries or an ALB certificate annotation." -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- end }}
