# fc
FC (Format converter) is a tool to convert different structured data types, like JSON and YAML.

    Usage: gofc INPUT_FILTER [FILTER [FILTER [...]]] OUTPUT_FILTER

    FILTER := -f <name> [-a arg1 [-a arg2 [...]]]

## Installation

    go install github.com/spirius/fc/cmd/gofc

## Examples

### Convert between JSON and YAML

    gofc -f j -f y < input.json > output.yml

and

    gofc -f j -f y < input.yml > output.json

### Render golang template using YAML as input context

    gofc -f y -f tpl -a template.tpl < input.tpl > rendered.txt

### Process YAML file using jq

    gofc -f y -f j < input.yml | jq '.' | gofc -f j -f y

## Input Filters

### JSON

    name = json | j

Parse input data as JSON.

### YAML

    name = yaml | yml | y

Parse input data as YAML.

## Filters

### Variable Files

    name = varfiles | v

This filter treats input data as map of variable names and files, reads each file as a structure and outputs new map of variables names and read structures.

Example: Merge YAML and JSON files, process with jq

    echo '{
        "jsondata": "file.json",
        "yamldata": "file.yml"
    }' | gofc -f json -f varfiles -f json | jq '.'

Output:

    {
        "var1": //parsed content of file1.json
        "var2": //parsed content of file2.json
    }

## Output Filters

### Tpl

    name = tpl | t
    args = template file

Renders the template file as golang template using input as template's context.
