{{/*
Expand the name of the chart.
*/}}
{{- define "claimctl.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "claimctl.fullname" -}}
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
{{- define "claimctl.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "claimctl.labels" -}}
helm.sh/chart: {{ include "claimctl.chart" . }}
{{ include "claimctl.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "claimctl.selectorLabels" -}}
app.kubernetes.io/name: {{ include "claimctl.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "claimctl.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "claimctl.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Resolve the PostgreSQL service host when the bundled database dependency is used.
Mirrors the postgres subchart fullname logic so name overrides are respected.
*/}}
{{- define "claimctl.postgresHost" -}}
{{- if .Values.postgres.fullnameOverride }}
{{- .Values.postgres.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default "postgres" .Values.postgres.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
The pod label value used by the postgres subchart for selector matching.
Mirrors the postgres subchart `postgres.name` helper.
*/}}
{{- define "claimctl.postgresName" -}}
{{- default "postgres" .Values.postgres.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
True when the ingress controller selector for NetworkPolicies is configured.
*/}}
{{- define "claimctl.ingressControllerConfigured" -}}
{{- if or (not (empty .Values.networkPolicy.ingressController.namespaceSelector)) (not (empty .Values.networkPolicy.ingressController.podSelector)) }}
true
{{- end }}
{{- end }}

{{/*
The Kubernetes secret holding the application database user/password/name.
*/}}
{{- define "claimctl.databaseSecretName" -}}
{{- if and .Values.postgres.enabled .Values.postgres.userDatabase.existingSecret }}
{{- .Values.postgres.userDatabase.existingSecret }}
{{- else if and (not .Values.postgres.enabled) .Values.db.existingSecret }}
{{- .Values.db.existingSecret }}
{{- else }}
{{- printf "%s-db-credentials" (include "claimctl.fullname" .) }}
{{- end }}
{{- end }}

{{/*
True when the chart itself manages the database credentials secret.
*/}}
{{- define "claimctl.isManagedDatabaseSecret" -}}
{{- if or (and .Values.postgres.enabled (not .Values.postgres.userDatabase.existingSecret)) (and (not .Values.postgres.enabled) (not .Values.db.existingSecret)) }}
true
{{- end }}
{{- end }}

{{/*
Key names within the resolved database secret.
*/}}
{{- define "claimctl.databaseUserKey" -}}
{{- if and .Values.postgres.enabled .Values.postgres.userDatabase.existingSecret }}
{{- .Values.postgres.userDatabase.user.secretKey | default "db-user" }}
{{- else if and (not .Values.postgres.enabled) .Values.db.existingSecret }}
{{- .Values.db.existingSecretUserKey | default "db-user" }}
{{- else }}
{{- "db-user" }}
{{- end }}
{{- end }}

{{- define "claimctl.databasePasswordKey" -}}
{{- if and .Values.postgres.enabled .Values.postgres.userDatabase.existingSecret }}
{{- .Values.postgres.userDatabase.password.secretKey | default "db-password" }}
{{- else if and (not .Values.postgres.enabled) .Values.db.existingSecret }}
{{- .Values.db.existingSecretPasswordKey | default "db-password" }}
{{- else }}
{{- "db-password" }}
{{- end }}
{{- end }}

{{- define "claimctl.databaseNameKey" -}}
{{- if and .Values.postgres.enabled .Values.postgres.userDatabase.existingSecret }}
{{- .Values.postgres.userDatabase.name.secretKey | default "db-name" }}
{{- else if and (not .Values.postgres.enabled) .Values.db.existingSecret }}
{{- .Values.db.existingSecretNameKey | default "db-name" }}
{{- else }}
{{- "db-name" }}
{{- end }}
{{- end }}
