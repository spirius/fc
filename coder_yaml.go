package fc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/juju/errors"
	"gopkg.in/yaml.v2"
)

type coderYAML struct{}

func (c *coderYAML) Initialize() error {
	return nil
}

func (c *coderYAML) Names() []string {
	return []string{"yaml", "yml", "y"}
}

var typeString = reflect.TypeOf("")

// normalize recursively converts
// map[interface{}]interface{} to map[string]interface{}
func (c coderYAML) normalize(in reflect.Value) reflect.Value {
	elem := in
	kind := in.Kind()
	typ := in.Type()
	if kind == reflect.Interface || kind == reflect.Ptr {
		elem = in.Elem()

		if !elem.IsValid() {
			return in
		}

		kind = elem.Kind()
		typ = elem.Type()
	}

	if kind == reflect.Map {
		keyKind := typ.Key().Kind()
		if keyKind != reflect.String {
			nmap := reflect.MakeMapWithSize(reflect.MapOf(typeString, typ.Elem()), elem.Len())
			for _, key := range elem.MapKeys() {
				nmap.SetMapIndex(reflect.ValueOf(key.Elem().String()), c.normalize(elem.MapIndex(key)))
			}
			return nmap
		}
		for _, key := range elem.MapKeys() {
			elem.SetMapIndex(key, c.normalize(elem.MapIndex(key)))
		}
	} else if kind == reflect.Slice && typ.Elem().Kind() == reflect.Interface {
		for i := 0; i < elem.Len(); i++ {
			v := elem.Index(i)
			v.Set(c.normalize(v))
		}
	}

	return in
}

func (c *coderYAML) Decode(in io.Reader, args []string) (interface{}, interface{}, error) {
	if len(args) > 0 {
		return nil, nil, errors.Trace(ArgumentError{error: fmt.Sprintf("YAML: unexpected input argument '%s', no arguments expected", args[0])})
	}

	data, err := ioutil.ReadAll(in)

	if err != nil {
		return nil, nil, err
	}

	var out interface{}
	if err = yaml.Unmarshal(data, &out); err != nil {
		return nil, nil, err
	}

	return c.normalize(reflect.Indirect(reflect.ValueOf(out))).Interface(), nil, nil
}

func (c *coderYAML) Encode(out io.Writer, in interface{}, metadata interface{}, args []string) error {
	if len(args) > 0 {
		return errors.Trace(ArgumentError{error: fmt.Sprintf("YAML: unexpected output argument '%s', no arguments expected", args[0])})
	}

	data, err := yaml.Marshal(in)
	if err != nil {
		return errors.Annotatef(err, "YAML: cannot marshal YAML")
	}

	_, err = io.Copy(out, bytes.NewReader(data))
	return errors.Annotatef(err, "YAML: cannot write")
}
