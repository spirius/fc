package fc

import (
	"fmt"
	"os"
	"path/filepath"
)

type FilterVarFiles struct {
	fc *FC
}

func (f FilterVarFiles) Filter(input interface{}, args ...string) (interface{}, error) {
	var (
		patterns map[string]interface{}
		ok       bool
		out      = make(map[string]interface{})
	)

	if patterns, ok = input.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("filter varfiles expectes map of file patterns as input, got something else")
	}

	for varName, p := range patterns {
		pattern := p.(string)

		files, err := filepath.Glob(pattern)

		if err != nil {
			return nil, err
		}

		var transform = true

		// BUG(not really consistent check, filename could actually contain '*' or '?')
		if len(files) == 1 && files[0] == filepath.Clean(pattern) {
			transform = false
		}

		var entries []interface{}

		for _, filename := range files {
			ext := filepath.Ext(filename)

			filter, err := f.fc.GetInputFilter(ext[1:])

			if err != nil {
				return nil, err
			}

			file, err := os.Open(filename)

			if err != nil {
				return nil, err
			}

			var res interface{}

			if err = filter.Input(file, &res); err != nil {
				return nil, err
			}

			if list, ok := res.([]interface{}); ok && transform {
				entries = append(entries, list...)
			} else {
				entries = append(entries, res)
			}
		}

		if !transform && len(entries) > 0 {
			out[varName] = entries[0]
		} else {
			out[varName] = entries
		}
	}

	return out, nil
}

const filterVarFilesDescription = `Filter takes as input map of filename patterns and outputs map containing contnent of each file.`

func (FilterVarFiles) Description() string {
	return filterVarFilesDescription
}

func init() {
	DefaultFC.AddFilter(&FilterVarFiles{DefaultFC}, "varfiles", "v")
}
