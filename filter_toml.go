package fc

import (
	"github.com/BurntSushi/toml"
	"io"
)

type FilterTOML struct {
	baseFilter
}

func (FilterTOML) Input(in io.Reader, out interface{}, args ...string) error {
	_, err := toml.DecodeReader(in, out)
	return err
}

func (FilterTOML) Output(out io.Writer, in interface{}, args ...string) error {
	return toml.NewEncoder(out).Encode(in)
}

const filterTOMLDescription = `Input/Output filter for TOML, args: none`

func (FilterTOML) Description() string {
	return filterTOMLDescription
}

func init() {
	f := &FilterTOML{}
	DefaultFC.AddInputFilter(f, "toml")
	DefaultFC.AddOutputFilter(f, "toml")
}
