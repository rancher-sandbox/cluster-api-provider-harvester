{{- define "caphv.name" -}}
caphv
{{- end }}

{{- define "caphv.fullname" -}}
caphv-controller-manager
{{- end }}

{{- define "caphv.namespace" -}}
{{ .Release.Namespace | default .Values.namespace }}
{{- end }}

{{- define "caphv.labels" -}}
control-plane: controller-manager
cluster.x-k8s.io/provider: infrastructure-harvester
cluster.x-k8s.io/v1beta1: v1alpha1
app.kubernetes.io/name: {{ include "caphv.name" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "caphv.selectorLabels" -}}
control-plane: controller-manager
{{- end }}
