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

type fileInfo struct {
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

func (f *FilterTPL) loadPatternS3(bucket, pattern string, filesInfo map[string]interface{}) (entries []interface{}, err error) {
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

	for key, obj := range files {
		res, err := f.parseToStruct(key, obj.Body)

		if err != nil {
			return nil, err
		}

		infoObj := &fileInfo{}

		if obj.VersionId != nil {
			infoObj.VersionId = *obj.VersionId
		}

		if list, ok := res.([]interface{}); ok {
			for _, e := range list {
				filesInfo[strconv.Itoa(len(entries))] = infoObj
				entries = append(entries, e)
			}
		} else {
			filesInfo[strconv.Itoa(len(entries))] = infoObj
			entries = append(entries, res)
		}
	}

	return entries, nil
}

func (f *FilterTPL) loadFile(urlStr string) (interface{}, error) {
	urlInfo, err := url.Parse(urlStr)

	if err != nil {
		return nil, err
	}

	if urlInfo.Scheme == "s3" {
		if f.s3 == nil {
			f.s3, err = NewS3Downloader()
			if err != nil {
				return nil, err
			}
		}

		parts := strings.SplitN(urlInfo.Path, ":", 2)

		var versionId string

		if len(parts) > 1 {
			versionId = parts[1]
		}

		obj, err := f.s3.DownloadFileVersion(urlInfo.Host, parts[0][1:], versionId)

		if err != nil {
			return nil, err
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

func (f *FilterTPL) loadPattern(urlStr string, filesInfo map[string]interface{}) (entries []interface{}, err error) {
	urlInfo, err := url.Parse(urlStr)

	if err != nil {
		return nil, err
	}

	if urlInfo.Scheme == "s3" {
		return f.loadPatternS3(urlInfo.Host, urlInfo.Path[1:], filesInfo)
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
