package fc

import (
	"io"
)

type FilterNULL struct {
	baseFilter
}

func (FilterNULL) Input(in io.Reader, out interface{}, args ...string) error {
	out = nil
	return nil
}

const filterNULLDescription = `Null input filter`

func (FilterNULL) Description() string {
	return filterNULLDescription
}

func init() {
	f := &FilterNULL{}
	DefaultFC.AddInputFilter(f, "n")
}
