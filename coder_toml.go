package fc

import (
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
)

type coderTOML struct{}

func (c *coderTOML) Initialize() error {
	return nil
}

func (c *coderTOML) Names() []string {
	return []string{"toml", "t"}
}

func (c *coderTOML) Decode(in io.Reader, args []string) (interface{}, interface{}, error) {
	if len(args) > 0 {
		return nil, nil, errors.Trace(ArgumentError{error: fmt.Sprintf("TOML: invalid input argument '%s', no arguments expected", args[0])})
	}
	var out interface{}
	if _, err := toml.DecodeReader(in, &out); err != nil {
		return nil, nil, errors.Annotatef(err, "cannot decode TOML")
	}
	return out, nil, nil
}

func (c *coderTOML) Encode(out io.Writer, in interface{}, metadata interface{}, args []string) error {
	if len(args) > 0 {
		return errors.Trace(ArgumentError{error: fmt.Sprintf("TOML: invalid output argument '%s', no arguments expected", args[0])})
	}
	return errors.Annotatef(toml.NewEncoder(out).Encode(in), "TOML: cannot parse")
}
