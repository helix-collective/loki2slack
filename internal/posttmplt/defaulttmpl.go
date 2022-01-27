package posttmplt

const DefaultTemplate = `
{{define "message" -}}
{{- $expr := .Query | escapequotes -}}
{{- $left := printf ` + "`" + `["%d","%d","%s",{"expr":"%s"}]` + "`" + `
        .EntryTimestamp .EntryTimestamp .LokiDataSource $expr
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
