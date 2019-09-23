{{ range $k, $v := $.list -}}
{{ $k }}:{{ $v }}
{{ end -}}
{{ include "./subdir/include.tpl" $.map }}
{{ include "./include.tpl" $ }}