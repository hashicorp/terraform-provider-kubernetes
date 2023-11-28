{{- if .ElementType -}}
  {{- if eq .AttributeType "ListAttribute" -}}
    {{ .FieldName }} []types.{{ .ElementType }} `tfsdk:"{{ .AttributeName }}" manifest:"{{ .ManifestFieldName }}"`
  {{- else if eq .AttributeType "MapAttribute" -}}
    {{ .FieldName }} map[string]types.{{ .ElementType }} `tfsdk:"{{ .AttributeName }}" manifest:"{{ .ManifestFieldName }}"`
  {{- end -}}
{{- else if .NestedFields -}}
  {{ .FieldName }} {{ if eq .AttributeType "ListNestedAttribute" -}}[]{{- end -}}struct{
    {{ .NestedFields }}
  } `tfsdk:"{{ .AttributeName }}" manifest:"{{ .ManifestFieldName }}"`
{{- else -}}
  {{ .FieldName }} types.{{ .Type }} `tfsdk:"{{ .AttributeName }}" manifest:"{{ .ManifestFieldName }}"`
{{- end -}}