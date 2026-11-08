{{/*
Fail-close validation of required configuration. Included from the backend
deployment so misconfigured releases are rejected before anything is applied.
*/}}
{{- define "claimctl.assertValues" -}}
{{- if not .Values.postgres.enabled -}}
  {{- if not (or .Values.db.host .Values.db.existingSecret) -}}
    {{- fail "claimctl: db.host must be set when postgres.enabled is false" -}}
  {{- end -}}
  {{- if not (or .Values.db.name .Values.db.existingSecret) -}}
    {{- fail "claimctl: db.name or db.existingSecret must be set when postgres.enabled is false" -}}
  {{- end -}}
  {{- if not (or .Values.db.user .Values.db.existingSecret) -}}
    {{- fail "claimctl: db.user or db.existingSecret must be set when postgres.enabled is false" -}}
  {{- end -}}
  {{- if not (or .Values.db.password .Values.db.existingSecret) -}}
    {{- fail "claimctl: db.password or db.existingSecret must be set when postgres.enabled is false" -}}
  {{- end -}}
{{- else -}}
  {{- if not (or .Values.postgres.settings.superuserPassword.value .Values.postgres.settings.existingSecret) -}}
    {{- fail "claimctl: postgres.settings.superuserPassword.value or postgres.settings.existingSecret is required" -}}
  {{- end -}}
  {{- if not (or .Values.postgres.userDatabase.user.value .Values.postgres.userDatabase.existingSecret) -}}
    {{- fail "claimctl: postgres.userDatabase.user must be configured" -}}
  {{- end -}}
  {{- if not (or .Values.postgres.userDatabase.password.value .Values.postgres.userDatabase.existingSecret) -}}
    {{- fail "claimctl: postgres.userDatabase.password must be configured" -}}
  {{- end -}}
  {{- if not (or .Values.postgres.userDatabase.name.value .Values.postgres.userDatabase.existingSecret) -}}
    {{- fail "claimctl: postgres.userDatabase.name must be configured" -}}
  {{- end -}}
{{- end -}}
{{- if not (or .Values.appEncryption.existingSecret .Values.keyGeneration.enabled) -}}
  {{- fail "claimctl: appEncryption.existingSecret or keyGeneration.enabled is required so APP_ENCRYPTION_KEY stays stable across replicas and restarts" -}}
{{- end -}}
{{- if and .Values.keyGeneration.enabled (not .Values.appEncryption.existingSecret) (not .Values.serviceAccount.create) -}}
  {{- fail "claimctl: keyGeneration.enabled requires serviceAccount.create to be true" -}}
{{- end -}}
{{- end -}}