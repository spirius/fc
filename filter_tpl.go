package fc

import (
	"fmt"
	"github.com/Masterminds/sprig"
	"io"
	"io/ioutil"
	"text/template"
)

type FilterTPL struct {
}

func (_ FilterTPL) Output(output io.Writer, input interface{}, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("tpl filter requires filename")
	}

	content, err := ioutil.ReadFile(args[0])

	if err != nil {
		return err
	}

	tpl, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(string(content))

	if err != nil {
		return err
	}

	return tpl.Execute(output, input)
}

const filterTPLDescription = `Output filter for rendering golang templates, args: <filename>`

func (FilterTPL) Description() string {
	return filterTPLDescription
}

func init() {
	DefaultFC.AddOutputFilter(&FilterTPL{}, "tpl", "t")
}
