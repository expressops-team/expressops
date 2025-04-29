{{/*
Expand the name of the chart.
*/}}
# Define el nombre base de la aplicación (probablemente "expressops-chart").
{{- define "expressops-chart.name" -}} 
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
# Crea un nombre único para la release (ej: RELEASE_NAME-expressops-chart). Este es el que usarás para metadata.name en la mayoría de los recursos.
{{- define "expressops-chart.fullname" -}} 
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
 # Usado internamente para la etiqueta helm.sh/chart.
{{- define "expressops-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
 # Define el conjunto completo de etiquetas recomendadas para identificar los recursos. Incluye las etiquetas de selector. Lo usarás en metadata.labels de todos tus recursos.
{{- define "expressops-chart.labels" -}}
helm.sh/chart: {{ include "expressops-chart.chart" . }}
{{ include "expressops-chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
# Define el conjunto mínimo de etiquetas para que los selectores funcionen (Deployment -> Pod, Service -> Pod).
{{- define "expressops-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "expressops-chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "expressops-chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "expressops-chart.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
