package fc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/juju/errors"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type coderHCL struct{}

func (c *coderHCL) Names() []string {
	return []string{"hcl", "h"}
}

func (c *coderHCL) Initialize() error {
	return nil
}

func (c *coderHCL) decodeBody(body *hclsyntax.Body) (attribs map[string]interface{}, blocks []interface{}, err error) {
	type hclValue struct {
		Value interface{}
		Type  interface{}
	}

	attribs = make(map[string]interface{})
	for _, attr := range body.Attributes {
		val, _ := attr.Expr.Value(nil)
		clean, err := cty.Transform(val, func(path cty.Path, value cty.Value) (cty.Value, error) {
			return cty.UnknownAsNull(value), nil
		})
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		js, err := ctyjson.Marshal(clean, cty.DynamicPseudoType)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		var r hclValue
		if err = json.Unmarshal(js, &r); err != nil {
			return nil, nil, errors.Trace(err)
		}

		attribs[attr.Name] = r.Value
	}

	blocks = make([]interface{}, 0)
	for _, block := range body.Blocks {
		b := map[string]interface{}{
			"type":   block.Type,
			"labels": block.Labels,
		}
		if block.Body != nil {
			b["attributes"], b["blocks"], err = c.decodeBody(block.Body)
			if err != nil {
				return nil, nil, errors.Trace(err)
			}
		}
		blocks = append(blocks, b)
	}

	return attribs, blocks, nil
}

func (c *coderHCL) Decode(in io.Reader, args []string) (interface{}, interface{}, error) {
	if len(args) > 0 {
		return nil, nil, errors.Trace(ArgumentError{error: fmt.Sprintf("HCL: invalid input argument '%s', no arguments expected", args[0])})
	}

	content, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "HCL: cannot read input data")
	}

	parser := hclparse.NewParser()
	file, diag := parser.ParseHCL(content, "")
	if diag.HasErrors() {
		return nil, nil, errors.Trace(diag)
	}

	attribs, metadata, err := c.decodeBody(file.Body.(*hclsyntax.Body))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return attribs, metadata, nil
}

func (c *coderHCL) Encode(out io.Writer, in interface{}, metadata interface{}, args []string) error {
	if len(args) > 0 {
		return errors.Trace(ArgumentError{error: fmt.Sprintf("HCL: unexpected output argument '%s', no arguments expected", args[0])})
	}

	jsonContent, err := json.Marshal(in)
	if err != nil {
		return errors.Annotatef(err, "HCL: cannot convert input to JSON")
	}

	parser := hclparse.NewParser()
	file, diag := parser.ParseJSON(jsonContent, "")
	if diag.HasErrors() {
		return errors.Trace(diag)
	}
	attrs, diag := file.Body.JustAttributes()
	if diag.HasErrors() {
		return errors.Trace(diag)
	}

	resFile := hclwrite.NewEmptyFile()
	resBody := resFile.Body()
	for _, attr := range attrs {
		val, diag := attr.Expr.Value(nil)
		if diag.HasErrors() {
			return errors.Trace(diag)
		}
		resBody.SetAttributeValue(attr.Name, val)
	}
	_, err = resFile.WriteTo(out)

	return errors.Trace(err)
}
