package fc

import (
	"encoding/json"
	"io"
)

type FilterJSON struct {
	baseFilter
}

func (FilterJSON) Input(in io.Reader, out interface{}, args ...string) error {
	return json.NewDecoder(in).Decode(out)
}

func (FilterJSON) Output(out io.Writer, in interface{}, args ...string) error {
	return json.NewEncoder(out).Encode(in)
}

const filterJSONDescription = `Input/Output filter for JSON (JavaScript Object Notation), args: none`

func (FilterJSON) Description() string {
	return filterJSONDescription
}

func init() {
	f := &FilterJSON{}
	DefaultFC.AddInputFilter(f, "json", "j")
	DefaultFC.AddOutputFilter(f, "json", "j")
}
