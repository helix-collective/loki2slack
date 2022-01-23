package posttmplt

const DefaultTemplate = `
{{define "message" -}}
{{- $left := printf ` + "`" + `["%d","%d","%s",{"expr":"{env=\"%s\"}"}]` + "`" + `
        .EntryTimestamp .EntryTimestamp .LokiDataSource .Labels.env
    | urlquery
-}}
{{- $lokiLink := printf "%[1]s/explore?left=%[2]s" .GrafanaUrl $left -}}
{{- printf "<%s|Grafana Link>" $lokiLink}}
{{- range $key, $value := .Labels }}
{{ $key }} = {{ $value }}
{{- end }}
{{- end}}

{{define "json_attachment" -}}
{{- range $key, $value := .Line -}}
{{ $key }}: {{ $value }}
{{end -}}
{{- end}}

{{define "txt_attachment" -}}
{{.Line}}
{{- end}}
`
