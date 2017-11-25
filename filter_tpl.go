package fc

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"io"
	"io/ioutil"
	"net"
	"text/template"
)

type FilterTPL struct {
	fc      *FC
	funcMap map[string]interface{}
}

func cidr_Contains(cidr_addr, ip_addr string) (r bool, err error) {
	_, cidr, err := net.ParseCIDR(cidr_addr)

	if err != nil {
		return false, err
	}

	return cidr.Contains(net.ParseIP(ip_addr)), nil
}

func (f FilterTPL) Output(output io.Writer, input interface{}, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("tpl filter requires filename")
	}

	content, err := ioutil.ReadFile(args[0])

	if err != nil {
		return err
	}

	for filterName, outputFilter := range f.fc.outputFilters {
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

	tpl, err := template.New("").Funcs(f.funcMap).Parse(string(content))

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

func NewFilterTPL() (r *FilterTPL) {
	r = &FilterTPL{}

	r.funcMap = sprig.TxtFuncMap()
	r.funcMap["cidr_contains"] = cidr_Contains
	return
}

func init() {
	DefaultFC.AddOutputFilter(NewFilterTPL(), "tpl", "t")
}
