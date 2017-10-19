package fc

import (
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
)

type FilterYAML struct {
}

func (FilterYAML) Input(in io.Reader, out interface{}, args ...string) error {
	data, err := ioutil.ReadAll(in)

	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, out)
}

func (FilterYAML) Output(out io.Writer, in interface{}, args ...string) error {
	data, err := yaml.Marshal(in)

	if err != nil {
		return err
	}

	size := len(data)
	left := size

	for left > 0 {
		n, err := out.Write(data[size-left:])

		if err != nil {
			return err
		}

		left -= n
	}

	return nil
}

const filterYAMLDescription = `Input/Output filter for YAML (Yet Another Markup Language)`

func (FilterYAML) Description() string {
	return filterYAMLDescription
}

func init() {
	f := &FilterYAML{}
	DefaultFC.AddInputFilter(f, "yaml", "yml", "y")
	DefaultFC.AddOutputFilter(f, "yaml", "yml", "y")
}
