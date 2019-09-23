package fc

import (
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/juju/errors"
)

type importer struct {
	recoder *Recoder
	s3      s3iface.S3API
}

type importOpts struct {
	raw      bool
	nofail   bool
	pattern  bool
	metadata bool
}

func newImporter(r *Recoder, s3 s3iface.S3API) *importer {
	return &importer{recoder: r, s3: s3}
}

func (t *importer) importURL(fileURL string, opts importOpts) (interface{}, error) {
	urlInfo, err := url.Parse(fileURL)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot import, invalid URL '%s'", fileURL)
	}

	switch urlInfo.Scheme {
	case "file", "":
		path := urlInfo.Host + urlInfo.Path
		if opts.pattern {
			return t.importFiles(path, opts)
		}
		return t.importFile(path, opts)
	case "s3":
		if opts.pattern {
			return nil, errors.Errorf("pattern option is not supported for s3 yet")
		}
		return t.importS3Object(urlInfo, opts)
	}

	return nil, errors.Errorf("cannot import, unknown URL scheme '%s' in '%s'", urlInfo.Scheme, fileURL)
}

func (t *importer) parseBody(fileURL string, file io.ReadCloser, opts importOpts) (interface{}, interface{}, error) {
	defer file.Close()

	if opts.raw {
		body, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, nil, errors.Annotatef(err, "cannot read imported file '%s'", fileURL)
		}
		return string(body), nil, nil
	}

	ext := filepath.Ext(fileURL)[1:]
	decoder, ok := t.recoder.Decoders[ext]
	if !ok {
		return nil, nil, errors.Errorf("unknown file extension '%s', cannot parse file '%s'", ext, fileURL)
	}

	res, metadata, err := decoder.Decode(file, nil)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "cannot parse imported file '%s'", fileURL)
	}

	return res, metadata, nil
}

func (t *importer) importFile(path string, opts importOpts) (res interface{}, err error) {
	var metadata = map[string]interface{}{
		"url": path,
	}
	defer func() {
		if opts.nofail && err != nil {
			metadata["error"] = err
			err = nil
		}
		if opts.nofail || opts.metadata {
			res = metadata
		} else {
			res = metadata["body"]
		}
	}()

	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot open import file '%s'", path)
	}

	metadata["body"], metadata["metadata"], err = t.parseBody(path, file, opts)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot parse imported file '%s'", path)
	}

	return
}

func (t *importer) importFiles(pattern string, opts importOpts) (entries []interface{}, err error) {
	files, err := filepath.Glob(pattern)

	if err != nil {
		return nil, errors.Annotatef(err, "import failed, cannot list files")
	}

	for _, path := range files {
		res, err := t.importFile(path, opts)
		if err != nil {
			return nil, errors.Annotatef(err, "import failed, cannot import file '%s'", path)
		}
		entries = append(entries, res)
	}

	return entries, nil
}

func (t *importer) importS3Object(urlInfo *url.URL, opts importOpts) (res interface{}, err error) {
	var (
		bucket  = urlInfo.Host
		key     = urlInfo.Path[1:]
		version = urlInfo.Query().Get("versionId")
	)
	var metadata = map[string]interface{}{
		"url":     urlInfo.String(),
		"bucket":  bucket,
		"key":     key,
		"version": version,
	}
	defer func() {
		if opts.nofail && err != nil {
			metadata["error"] = err
			err = nil
		}
		if opts.nofail || opts.metadata {
			res = metadata
		} else {
			res = metadata["body"]
		}
	}()

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	obj, err := t.s3.GetObject(input)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot import s3 file '%s'", urlInfo)
	}

	metadata["body"], metadata["metadata"], err = t.parseBody(key, obj.Body, opts)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot parse imported file '%s'", urlInfo)
	}

	return
}
