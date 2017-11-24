package fc

import (
	"fmt"
	"github.com/Masterminds/sprig"
	"io"
	"io/ioutil"
	"net"
	"text/template"
)

type FilterTPL struct {
}

var funcMap = sprig.TxtFuncMap()

func ipIn(cidr, ipaddr string) (r bool, err error) {
	_, subnet, err := net.ParseCIDR(cidr)

	if err != nil {
		return false, err
	}

	return subnet.Contains(net.ParseIP(ipaddr)), nil
}

func init() {
	funcMap["ip_in"] = ipIn
}

func (_ FilterTPL) Output(output io.Writer, input interface{}, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("tpl filter requires filename")
	}

	content, err := ioutil.ReadFile(args[0])

	if err != nil {
		return err
	}

	tpl, err := template.New("").Funcs(funcMap).Parse(string(content))

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
