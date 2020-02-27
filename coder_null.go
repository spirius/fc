package fc

import (
	"io"
)

type coderNULL struct{}

func (c *coderNULL) Initialize() error {
	return nil
}

func (c *coderNULL) Names() []string {
	return []string{"null", "n"}
}

func (c *coderNULL) Decode(in io.Reader, args []string) (interface{}, interface{}, error) {
	return nil, nil, nil
}
