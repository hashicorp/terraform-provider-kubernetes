MarkdownDescription: `{{ .Description }}`,
{{- if .ElementType }}
ElementType: types.{{ .ElementType }},
{{- end }}
{{- if .Required }}
Required: true,
{{- else }}
Optional: true,
{{- end }}
{{- if .Computed }}
Computed: true,
{{- end }}
{{- if .NestedAttributes }}
  {{- if eq .AttributeType "ListNestedAttribute" }}
  NestedObject: schema.NestedAttributeObject{
    {{ .NestedAttributes }}
  },
  {{- else }}
  {{ .NestedAttributes }}
  {{- end }}
{{- end }}