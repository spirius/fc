package fc

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/juju/errors"
)

type coderJSON struct{}

func (c *coderJSON) Initialize() error {
	return nil
}

func (c *coderJSON) Names() []string {
	return []string{"json", "j"}
}

func (c *coderJSON) Decode(in io.Reader, args []string) (interface{}, interface{}, error) {
	if len(args) > 0 {
		return nil, nil, errors.Trace(ArgumentError{error: fmt.Sprintf("JSON: invalid input argument '%s', no arguments expected", args[0])})
	}
	var out interface{}
	if err := json.NewDecoder(in).Decode(&out); err != nil {
		return nil, nil, errors.Annotatef(err, "cannot decode JSON")
	}
	return out, nil, nil
}

func (c *coderJSON) Encode(out io.Writer, in interface{}, metadata interface{}, args []string) error {
	encoder := json.NewEncoder(out)
	if len(args) == 1 && args[0] == "pretty" {
		encoder.SetIndent("", "  ")
	} else if len(args) > 0 {
		return errors.Trace(ArgumentError{error: fmt.Sprintf("JSON: invalid output argument '%s', supported arguments: 'pretty'", args[0])})
	}
	return encoder.Encode(in)
}
