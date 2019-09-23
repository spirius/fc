package fc

import (
	"bytes"
	"encoding/json"
	_ "fmt"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/require"
)

func TestImporterFile_basic(t *testing.T) {
	require := require.New(t)
	importer := newImporter(DefaultRecoder, nil)

	res, err := importer.importURL("file://testdata/file1.json", importOpts{})
	require.NoError(err)
	js, err := json.Marshal(res)
	require.NoError(err)
	require.JSONEq(`{"list":[1,2,3],"map":{"key":"value"}}`, string(js))

	res, err = importer.importURL("file://testdata/file1.json", importOpts{raw: true})
	require.NoError(err)
	require.Equal(res, `{
  "list": [1,2,3],
  "map": {
    "key": "value"
  }
}
`)
}

func TestImporterFile_metadata(t *testing.T) {
	require := require.New(t)
	importer := newImporter(DefaultRecoder, nil)

	res, err := importer.importURL("file://testdata/file1.json", importOpts{metadata: true})
	require.NoError(err)
	data, ok := res.(map[string]interface{})
	require.True(ok)
	require.Equal(data["url"], "testdata/file1.json")

	js, err := json.Marshal(data["body"])
	require.NoError(err)
	require.JSONEq(`{"list":[1,2,3],"map":{"key":"value"}}`, string(js))
}

func TestImporterFile_nofail(t *testing.T) {
	require := require.New(t)
	importer := newImporter(DefaultRecoder, nil)

	res, err := importer.importURL("file://testdata/file-not-found.json", importOpts{nofail: true})
	require.NoError(err)
	data, ok := res.(map[string]interface{})
	require.True(ok)
	require.Error(data["error"].(error))
}

func TestImporterFiles_basic(t *testing.T) {
	require := require.New(t)
	importer := newImporter(DefaultRecoder, nil)

	res, err := importer.importURL("file://testdata/import/basic/*.json", importOpts{pattern: true})
	require.NoError(err)
	list, ok := res.([]interface{})
	require.True(ok)
	require.Equal(2, len(list))
	require.Equal("file1", list[0].(map[string]interface{})["file"])
	require.Equal("file2", list[1].(map[string]interface{})["file"])
}

func TestImporterFiles_nofail(t *testing.T) {
	require := require.New(t)
	importer := newImporter(DefaultRecoder, nil)

	_, err := importer.importURL("file://testdata/import/error/*.json", importOpts{pattern: true})
	require.Error(err)

	res, err := importer.importURL("file://testdata/import/error/*.json", importOpts{pattern: true, nofail: true})
	require.NoError(err)
	list, ok := res.([]interface{})
	require.True(ok)
	require.Equal(3, len(list))

	for _, e := range list {
		data := e.(map[string]interface{})
		url := data["url"].(string)
		if url == "testdata/import/error/file1.json" || url == "testdata/import/error/file3.json" {
			require.Nil(data["error"])
		} else {
			require.Error(data["error"].(error))
		}
	}
}

type mockS3Client struct {
	s3iface.S3API
	putObject func(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
	getObject func(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

func (m *mockS3Client) PutObject(in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	if m.putObject == nil {
		panic("PutObject S3 mock function is not set")
	}
	return m.putObject(in)
}

func (m *mockS3Client) GetObject(in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if m.getObject == nil {
		panic("GetObject S3 mock function is not set")
	}
	return m.getObject(in)
}

var s3NoSuchKey = awserr.NewRequestFailure(awserr.New(s3.ErrCodeNoSuchKey, `The specified key does not exist.`, nil), 404, "id")

func TestImporterS3File_basic(t *testing.T) {
	s3client := &mockS3Client{}
	s3client.getObject = func(in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
		return &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewBufferString(testInput)),
		}, nil
	}
	importer := newImporter(DefaultRecoder, s3client)
	res, err := importer.importURL("s3://bucket/file.json", importOpts{})
	require.NoError(t, err)
	js, err := json.Marshal(res)
	require.NoError(t, err)
	require.JSONEq(t, testInput, string(js))

	s3client.getObject = func(in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
		if aws.StringValue(in.VersionId) != "0123456789" {
			return nil, s3NoSuchKey
		}
		return &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewBufferString(testInput)),
		}, nil
	}
	res, err = importer.importURL("s3://mybucket/file.json?versionId=0123456789", importOpts{metadata: true})
	require.NoError(t, err)
	data := res.(map[string]interface{})
	js, err = json.Marshal(data["body"])
	require.NoError(t, err)
	require.JSONEq(t, testInput, string(js))
	require.Equal(t, "mybucket", data["bucket"])
	require.Equal(t, "file.json", data["key"])
	require.Equal(t, "0123456789", data["version"])
}
