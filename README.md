Format converter (FC) is a structured data format converter and template rendering engine.

# Table of contents

* [Overview](#overview)
* [Download](#download)
* [Usage](#usage)
* [Templating](#templating)
  * [Additional template functions](#additional-template-functions)
    * [include](#include-path-input---string)
    * [decode_*](#decode_-data---map)
    * [encode_*](#encode_-data---map)
    * [import](#import-url-opts---map)
    * [metadata](#metadata---any)
    * [jq](#jq-expr-data---any)
* [Notes](#Notes)

# Overview

FC is a tool for converting structured data between different
representations, like JSON -> YAML or HCL -> TOML, etc. It also includes templating engine
based on [golang's built-in template language](https://golang.org/pkg/text/template/) and packaged with [sprig](http://masterminds.github.io/sprig/) extensions.

In essence gofc consists from decoder and encoder connected to each-other. It expects input data on **stdin** and outputs on **stdout**.

Supported input formats are: **JSON**, **YAML**, **TOML** and **HCL**.

Supported output formats are: **JSON**, **YAML**, **TOML**, **HCL** and **template**.

HCL2 format have types of constructs - 
[arguments](https://www.terraform.io/docs/configuration/syntax.html#arguments) and [blocks](https://www.terraform.io/docs/configuration/syntax.html#blocks). In gofc `arguments` are used as primary input stream, so conversion from HCL -> JSON will output only `arguments` and `blocks` will be ignored. `Blocks` are available in template engine via [metadata](#metadata---any) function.

# Download

gofc is provided as a static binary.
You can [download](https://github.com/spirius/fc/releases/latest)
it from releases page.

If you want to compile it, make sure
to install libjq and libonig devel packages.

# Usage

```
Usage:
gofc -i DECODER [ARG1, [...]] -o ENCODER [ARG1, [...]]
```

```
Options:
 -i            - input decoder
 -o            - output encoder
 -check-update - check if new version is available
 -self-update  - update to latest version

Supported coders:
json, j        - JSON decoder/encoder
yaml, yml, y   - YANL decoder/encoder
hcl, h         - HCL decoder/encoder, only Attributes are supported, blocks are ignored
toml, t        - TOML decoder/encoder
tpl            - template encoder
  ARG1          - template file path
```

**Convert from JSON to YAML**

```bash
$ echo '{"key":"value"}' | gofc -i j -o y
key: value
```

**Convert YAML file to JSON file**
```bash
$ gofc -i y -o j < input.yml > output.json
```

# Templating

Using gofc it is easy to render templates. You can use content with any of the supported input formats and pass it as a context object to templating engine.

Suppose you want to render some complex configuration file (nginx config) based on your own human-readable configuration files. We can store the human-readable part as HCL in some config file:

```hcl
// input.hcl
paths = {
  "/" = {
    upstream = "us1"
  }
  "/static/" = {
    upstream = "us2"
  }
}
```


```
# paths.conf.tpl
{{ range $path, $conf := $.paths }}
location {{ $path }} {
  proxy_pass http://{{ $conf.upstream }};
}
{{ end }}
```

We can use `input.hcl` to render `paths.conf.tpl` with following command

```
gofc < input.hcl -i hcl -o tpl paths.conf.tpl > paths.conf
```

And result will be rendered as 

```
location / {
  proxy_pass http://us1;
}

location /static/ {
  proxy_pass http://us2;
}
```

### Additional template functions

In addition to template [built-in functions](https://golang.org/pkg/text/template/#hdr-Functions) and [sprig extensions](http://masterminds.github.io/sprig), gofc adds following additional functions into templating engine.

#### `include $path $input -> string`
Renders template specified by `$path` using `$input` as template context. Includes are done relative to the current template file.

#### `decode_* $data -> map`
Decodes string `$data` into map. You can use any supported format instead of `*`.

For example: `decode_json $data`.

#### `encode_* $data -> map`
Encodes `$data` into string. You can use any supported format instead of `*`.

For example `encode_yaml $obj`.

#### `import $url $opts -> map`
Reads and optionally decodes content under `$url`. Supported schemes are `file://` and `s3://`. If scheme is not specified, `file://` will be used.

`$opts` is comma-separated string of options. Possible options are:

 * `raw` - return file content as string without parsing.

 * `metadata` - return file metadata as well. Changes return type to map:
 ```
 {
   "url":
   "body":

   // if scheme is s3
   "key":
   "bucket":
   "version":
 }
 ```
 * `nofail` - Enables `metadata` option and disabled failing of template generation in case of errors. If error occurred, it is stored in `error` field of the result.

 * `pattern` - treats path component of `$url` as [pattern](https://golang.org/pkg/path/filepath/#Match) and changes return type to list of files. If `nofail` or `metadata` options are enabled, they will be applied per-object in the result.

Examples:

Read a config file
```
$config := import config.yml
```

Read list of resources with file versions from S3 bucket.
```
$res := import s3://bucket/resources/*.yml "metadata,pattern"
```

#### `metadata -> any`

Get metadata of the input. Applicable only for HCL format. The blocks are returned as metadata.

#### `jq $expr $data -> any`

Run [jq](https://stedolan.github.io/jq/) filter on `$data`. Note, that only first value from
result will be returned.

Example
```
metadata | jq "reduce (.[] | select(.type == 'locals') | .attributes) as $i({};. + $i)" | toJson
```

If applied on
```hcl
locals {
  key1 = "var1"
}
locals {
  key2 = "var2"
  key3 = "var3"
}
```

Will produce following
```json
{
  "key1": "var1",
  "key2": "var2",
  "key3": "var3"
}
```

# Notes

* HCL and TOML are not supporting primitive types or arrays as root element.

```bash
echo 42 | gofc -i json -o yaml  # works as expected
echo 42 | gofc -i json -o hcl   # will fail
```
