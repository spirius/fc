package fc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/ashb/jqrepl/jq"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/juju/errors"
)

type coderTPL struct {
	funcMap  map[string]interface{}
	conv     *Recoder
	importer *importer
}

func newCoderTPL(conv *Recoder, s3client s3iface.S3API) *coderTPL {
	c := &coderTPL{}
	c.importer = newImporter(conv, s3client)
	c.conv = conv
	return c
}

func (c *coderTPL) Initialize() error {
	c.funcMap = sprig.TxtFuncMap()
	c.funcMap["include"] = c.tplFuncInclude
	c.funcMap["import"] = c.tplFuncImport
	c.funcMap["jq"] = func(p string, in interface{}) (interface{}, error) {
		libjq, err := jq.New()
		if err != nil {
			return nil, errors.Annotatef(err, "cannot initialize jq library")
		}
		defer libjq.Close()
		chanIn, chanOut, chanErr := libjq.Start(strings.Replace(p, "'", "\"", -1), jq.JvArray())
		inCopy, err := jq.JvFromInterface(in)
		if err != nil {
			return nil, errors.Annotatef(err, "cannot encode input data for jq")
		}

		var res interface{}
		for chanErr != nil && chanOut != nil {
			select {
			case e, ok := <-chanErr:
				if !ok {
					chanErr = nil
				} else {
					err = errors.Trace(e)
				}
			case o, ok := <-chanOut:
				if !ok {
					chanOut = nil
				} else if res == nil {
					res = o.ToGoVal()
				}
			case chanIn <- inCopy:
				close(chanIn)
				chanIn = nil
			}
		}
		return res, err
	}

	for n, f := range c.conv.Coders {
		name := n
		if _, ok := f.(Decoder); ok {
			c.funcMap["decode_"+name] = func(in string, args ...string) (interface{}, error) {
				data, _, err := c.conv.Decode(&Config{
					Decoder:     name,
					DecoderArgs: args,
					Input:       bytes.NewBufferString(in),
				})
				if err != nil {
					return nil, errors.Annotatef(err, "error while decoding %s", name)
				}
				return data, nil
			}
		}
		if _, ok := f.(Encoder); ok {
			c.funcMap["encode_"+name] = func(in map[string]interface{}, args ...string) (string, error) {
				var buf bytes.Buffer
				err := c.conv.Encode(&Config{
					Encoder:     name,
					EncoderArgs: args,
					Output:      &buf,
				}, in, nil)
				if err != nil {
					return "", errors.Annotatef(err, "error while encoding %s", name)
				}
				return buf.String(), nil
			}
		}
	}
	return nil
}

func (c *coderTPL) tplFuncImport(fileURL string, options ...string) (res interface{}, err error) {
	var opts importOpts

	for _, p := range strings.Split(strings.Join(options, ","), ",") {
		p = strings.TrimSpace(p)
		switch p {
		case "":
		case "raw":
			opts.raw = true
		case "nofail":
			opts.nofail = true
		case "pattern":
			opts.pattern = true
		case "metadata":
			opts.metadata = true
		default:
			return nil, errors.Errorf("unexpected import option '%s'", p)
		}
	}

	return c.importer.importURL(fileURL, opts)
}

func (c *coderTPL) tplFuncInclude(path string, ctx interface{}, metadata ...interface{}) (string, error) {
	buf, err := c.include(path, ctx, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	return buf.String(), nil
}

func (c *coderTPL) Names() []string {
	return []string{"tpl"}
}

func (c *coderTPL) Encode(out io.Writer, in interface{}, metadata interface{}, args []string) error {
	if len(args) != 1 {
		return errors.Trace(ArgumentError{error: fmt.Sprintf("tpl: expecting one argument: template file")})
	}
	buf, err := c.include(args[0], in, metadata)
	if err != nil {
		return errors.Annotatef(err, "tpl: error while parsing template")
	}
	_, err = io.Copy(out, buf)
	return errors.Annotatef(err, "tpl: cannot write")
}

func (c *coderTPL) newFuncMap(metadata interface{}) map[string]interface{} {
	funcMap := make(map[string]interface{})
	for k, v := range c.funcMap {
		funcMap[k] = v
	}
	funcMap["metadata"] = func() interface{} {
		return metadata
	}
	return funcMap
}

func (c *coderTPL) include(path string, ctx interface{}, metadata interface{}) (*bytes.Buffer, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Annotatef(err, "tpl: cannot get current directory")
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			panic(fmt.Sprintf("err: cannot chdir back to '%s', %s", cwd, err))
		}
	}()

	dir := filepath.Dir(path)

	if err = os.Chdir(dir); err != nil {
		return nil, errors.Annotatef(err, "tpl: cannot chdir to template directory '%s'", dir)
	}

	content, err := ioutil.ReadFile(filepath.Base(path))
	if err != nil {
		return nil, errors.Annotatef(err, "tpl: cannot read template '%s'", path)
	}

	tpl, err := template.New(filepath.Join(cwd, path)).Funcs(c.newFuncMap(metadata)).Parse(string(content))
	if err != nil {
		return nil, errors.Annotatef(err, "tpl: cannot parse template '%s'", path)
	}
	var buf bytes.Buffer
	if err = tpl.Execute(&buf, ctx); err != nil {
		return nil, errors.Annotatef(err, "tpl: cannot render template '%s'", path)
	}
	return &buf, nil
}
