package fc

import (
	"io"

	"encoding/json"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl/json/parser"
)

type FilterHCL struct {
	baseFilter
}

func (FilterHCL) Output(out io.Writer, in interface{}, args ...string) error {
	j, err := json.Marshal(in)

	if err != nil {
		return err
	}

	root, err := parser.Parse(j)

	if err != nil {
		return err
	}

	return printer.DefaultConfig.Fprint(out, root)
}

const filterHCLDescription = `Output filter to generate HCL (HashiCorp Configuration Language), args: none`

func (FilterHCL) Description() string {
	return filterHCLDescription
}

func init() {
	DefaultFC.AddOutputFilter(&FilterHCL{}, "hcl")
}
