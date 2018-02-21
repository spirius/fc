package fc

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"text/template"
)

type FilterTPL struct {
	fc       *FC
	funcMap  map[string]interface{}
	basepath string
}

func cidr_Contains(cidr_addr, ip_addr string) (r bool, err error) {
	_, cidr, err := net.ParseCIDR(cidr_addr)

	if err != nil {
		return false, err
	}

	return cidr.Contains(net.ParseIP(ip_addr)), nil
}

func (f FilterTPL) createTpl(content string) (*template.Template, error) {
	for filterName, outputFilter := range f.fc.OutputFilters {
		// copy to new variable, so that callback function will
		// have right reference to outputFilter
		var of = outputFilter
		f.funcMap["output_"+filterName] = func(input interface{}, args ...string) (_ string, err error) {
			var buffer bytes.Buffer
			if err = of.Output(&buffer, input, args...); err != nil {
				return
			}
			return buffer.String(), nil
		}
	}

	return template.New("").Funcs(f.funcMap).Parse(content)
}

func (f FilterTPL) Output(output io.Writer, input interface{}, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("tpl filter requires filename")
	}

	content, err := ioutil.ReadFile(args[0])

	if err != nil {
		return err
	}

	tpl, err := f.createTpl(string(content))

	if err != nil {
		return err
	}

	return tpl.Execute(output, input)
}

const filterTPLDescription = `Output filter for rendering golang templates, args: <filename>`

func (FilterTPL) Description() string {
	return filterTPLDescription
}

func (f *FilterTPL) setFC(fc *FC) {
	f.fc = fc
}

type TplImport struct {
	tpl *template.Template
}

func (t TplImport) Render(input interface{}) (string, error) {
	var buf bytes.Buffer

	if err := t.tpl.Execute(&buf, input); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (f FilterTPL) importTpl(filename string) (_ interface{}, err error) {
	if filepath.IsAbs(filename) && f.basepath != "" {
		if filename, err = filepath.Abs(f.basepath + "/" + filename); err != nil {
			return nil, err
		}
	}

	contentBytes, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	tpl, err := f.createTpl(string(contentBytes))

	if err != nil {
		return nil, err
	}

	return &TplImport{
		tpl: tpl,
	}, nil
}

func NewFilterTPL() (r *FilterTPL) {
	r = &FilterTPL{}

	r.funcMap = sprig.TxtFuncMap()
	r.funcMap["cidr_contains"] = cidr_Contains
	r.funcMap["import"] = func(filename string) (interface{}, error) {
		return r.importTpl(filename)
	}
	r.funcMap["render"] = func(content string, ctx interface{}) (interface{}, error) {
		tpl, err := r.createTpl(content)

		if err != nil {
			return nil, err
		}

		return TplImport{tpl: tpl}.Render(ctx)
	}

	r.basepath = os.Getenv("FC_TPL_BASEPATH")

	return
}

func init() {
	DefaultFC.AddOutputFilter(NewFilterTPL(), "tpl", "t")
}
