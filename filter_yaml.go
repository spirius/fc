package fc

import (
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"reflect"
)

type FilterYAML struct {
	baseFilter
}

var typeString = reflect.TypeOf("")

// normalize recursively converts
// map[interface{}]interface{} -> map[string]interface{}
func (f FilterYAML) normalize(in reflect.Value) reflect.Value {
	if in.Kind() != reflect.Interface {
		return in
	}

	elem := in.Elem()

	if !elem.IsValid() {
		return in
	}

	kind := elem.Kind()
	typ := elem.Type()

	if kind == reflect.Map && typ.Key().Kind() == reflect.Interface {
		nmap := reflect.MakeMapWithSize(reflect.MapOf(typeString, typ.Elem()), elem.Len())

		for _, key := range elem.MapKeys() {
			nmap.SetMapIndex(key.Elem(), f.normalize(elem.MapIndex(key)))
		}

		return nmap
	} else if kind == reflect.Slice && typ.Elem().Kind() == reflect.Interface {
		for i := 0; i < elem.Len(); i++ {
			v := elem.Index(i)
			v.Set(f.normalize(v))
		}
	}

	return in
}

func (f FilterYAML) Input(in io.Reader, out interface{}, args ...string) error {
	data, err := ioutil.ReadAll(in)

	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(data, out); err != nil {
		return err
	}

	outRef := reflect.Indirect(reflect.ValueOf(out))
	outRef.Set(f.normalize(outRef))

	return nil
}

func (FilterYAML) Output(out io.Writer, in interface{}, args ...string) error {
	data, err := yaml.Marshal(in)

	if err != nil {
		return err
	}

	size := len(data)
	left := size

	for left > 0 {
		n, err := out.Write(data[size-left:])

		if err != nil {
			return err
		}

		left -= n
	}

	return nil
}

const filterYAMLDescription = `Input/Output filter for YAML (Yet Another Markup Language)`

func (FilterYAML) Description() string {
	return filterYAMLDescription
}

func init() {
	f := &FilterYAML{}
	DefaultFC.AddInputFilter(f, "yaml", "yml", "y")
	DefaultFC.AddOutputFilter(f, "yaml", "yml", "y")
}
