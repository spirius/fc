package fc

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/juju/errors"
)

type FilterTPL struct {
	fc       *FC
	funcMap  map[string]interface{}
	basepath string
	s3       *S3Downloader
}

func cidr_Contains(cidr_addr, ip_addr string) (r bool, err error) {
	_, cidr, err := net.ParseCIDR(cidr_addr)

	if err != nil {
		return false, err
	}

	return cidr.Contains(net.ParseIP(ip_addr)), nil
}

func (f FilterTPL) createTpl(content string) (*template.Template, error) {
	funcMap := make(map[string]interface{}, len(f.funcMap))
	for k, v := range funcMap {
		funcMap[k] = v
	}
	for filterName, outputFilter := range f.fc.OutputFilters {
		// copy to new variable, so that callback function will
		// have right reference to outputFilter
		var of = outputFilter
		funcMap["output_"+filterName] = func(input interface{}, args ...string) (_ string, err error) {
			var buffer bytes.Buffer
			if err = of.Output(&buffer, input, args...); err != nil {
				return
			}
			return buffer.String(), nil
		}
	}

	for filterName, inputFilter := range f.fc.InputFilters {
		// copy to new variable, so that callback function will
		// have right reference to inputFilter
		var filter = inputFilter
		funcMap["input_"+filterName] = func(input string, args ...string) (output interface{}, err error) {
			err = filter.Input(bytes.NewBufferString(input), &output, args...)
			return
		}
	}

	return template.New("").Funcs(funcMap).Parse(content)
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
	dir string
}

func (t TplImport) Render(input interface{}) (string, error) {
	var buf bytes.Buffer
	var cwd string
	var err error

	chdir := t.dir != "" && t.dir != "."

	if chdir {

		// Remember cwd
		cwd, err = os.Getwd()
		if err != nil {
			return "", errors.Trace(err)
		}

		// change dir to template's dir
		if err = os.Chdir(t.dir); err != nil {
			return "", errors.Trace(err)
		}
	}

	// Execute the template
	if err = t.tpl.Execute(&buf, input); err != nil {
		return "", errors.Trace(err)
	}

	if chdir {
		// Recover cwd
		if err = os.Chdir(cwd); err != nil {
			return "", errors.Trace(err)
		}
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
		dir: filepath.Dir(filename),
	}, nil
}

type fileInfo struct {
	Bucket    string
	Key       string
	VersionId string
}

func (f *FilterTPL) parseToStruct(filename string, body io.Reader) (res interface{}, err error) {
	ext := filepath.Ext(filename)

	filter, err := f.fc.GetInputFilter(ext[1:])

	if err != nil {
		return nil, err
	}

	if err = filter.Input(body, &res); err != nil {
		return
	}

	return
}

func (f *FilterTPL) loadPatternS3(bucket, pattern string, args ...map[string]interface{}) (entries []interface{}, err error) {
	if f.s3 == nil {
		f.s3, err = NewS3Downloader()
		if err != nil {
			return nil, err
		}
	}

	files, err := f.s3.DownloadFilesByPattern(bucket, pattern)

	if err != nil {
		return nil, err
	}

	var filesInfo map[string]interface{}

	if len(args) > 0 {
		filesInfo = args[0]
	}

	for key, obj := range files {
		res, err := f.parseToStruct(key, obj.Body)

		if err != nil {
			return nil, err
		}

		infoObj := &fileInfo{}

		if obj.VersionId != nil {
			infoObj.VersionId = *obj.VersionId
		}
		infoObj.Bucket = bucket
		infoObj.Key = key

		if list, ok := res.([]interface{}); ok {
			for _, e := range list {
				if filesInfo != nil {
					filesInfo[strconv.Itoa(len(entries))] = infoObj
				}
				entries = append(entries, e)
			}
		} else {
			if filesInfo != nil {
				filesInfo[strconv.Itoa(len(entries))] = infoObj
			}
			entries = append(entries, res)
		}
	}

	return entries, nil
}

func (f *FilterTPL) loadFile(urlStr string, args ...map[string]interface{}) (interface{}, error) {
	urlInfo, err := url.Parse(urlStr)

	if err != nil {
		return nil, err
	}

	var info map[string]interface{}

	if len(args) > 0 {
		info = args[0]
	}

	if urlInfo.Scheme == "s3" {
		if f.s3 == nil {
			f.s3, err = NewS3Downloader()
			if err != nil {
				return nil, err
			}
		}

		parts := strings.SplitN(urlInfo.Path, ":", 2)

		var (
			versionId string
			key       string
		)

		if len(parts) > 1 {
			versionId = parts[1]
		}

		key = parts[0][1:]

		obj, err := f.s3.DownloadFileVersion(urlInfo.Host, key, versionId)

		if err != nil {
			return nil, err
		}

		if info != nil {
			info["0"] = &fileInfo{
				Bucket:    urlInfo.Host,
				Key:       key,
				VersionId: versionId,
			}
		}

		return f.parseToStruct(parts[0], obj.Body)
	} else if urlInfo.Scheme == "file" || urlInfo.Scheme == "" {
		filename := urlInfo.Host + urlInfo.Path

		if filepath.IsAbs(filename) && f.basepath != "" {
			if filename, err = filepath.Abs(f.basepath + "/" + filename); err != nil {
				return nil, err
			}
		}

		file, err := os.Open(filename)

		if err != nil {
			return nil, err
		}

		return f.parseToStruct(filename, file)
	} else {
		return nil, fmt.Errorf("Unknown scheme '%s'", urlInfo.Scheme)
	}
}

func (f *FilterTPL) loadPattern(urlStr string, args ...map[string]interface{}) (entries []interface{}, err error) {
	urlInfo, err := url.Parse(urlStr)

	if err != nil {
		return nil, err
	}

	var filesInfo map[string]interface{}

	if len(args) > 0 {
		filesInfo = args[0]
	}

	if urlInfo.Scheme == "s3" {
		return f.loadPatternS3(urlInfo.Host, urlInfo.Path[1:], filesInfo)
	} else if urlInfo.Scheme == "file" || urlInfo.Scheme == "" {
		pattern := urlInfo.Host + urlInfo.Path

		if filepath.IsAbs(pattern) && f.basepath != "" {
			if pattern, err = filepath.Abs(f.basepath + "/" + pattern); err != nil {
				return nil, err
			}
		}

		files, err := filepath.Glob(pattern)

		if err != nil {
			return nil, err
		}

		for _, filename := range files {
			file, err := os.Open(filename)

			if err != nil {
				return nil, err
			}

			res, err := f.parseToStruct(filename, file)

			if err != nil {
				return nil, err
			}

			if list, ok := res.([]interface{}); ok {
				entries = append(entries, list...)
			} else {
				entries = append(entries, res)
			}
		}

		return entries, nil
	} else {
		return nil, fmt.Errorf("Unknown scheme '%s'", urlInfo.Scheme)
	}
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

	r.funcMap["loadPattern"] = r.loadPattern
	r.funcMap["load"] = r.loadFile

	return
}

func init() {
	DefaultFC.AddOutputFilter(NewFilterTPL(), "tpl", "t")
}
