Attributes: map[string]schema.Attribute{
    {{- range $val := . }}
    "{{- $val.Name }}": schema.{{ $val.AttributeType }}{ 
        {{ $val }} 
    },
    {{- end }}
},