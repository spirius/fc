{{ range $k, $v := $ -}}
{{ $k }}:{{ $v }}
{{ end -}}
{{ include "../include.tpl" $ }}