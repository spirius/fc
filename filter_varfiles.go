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

	for varName, pattern := range patterns {
		files, err := filepath.Glob(pattern.(string))

		if err != nil {
			return nil, err
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

			if list, ok := res.([]interface{}); ok {
				entries = append(entries, list...)
			} else {
				entries = append(entries, res)
			}
		}

		out[varName] = entries
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
