package fc

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"path/filepath"
	"strings"
	"sync"
)

// S3Downloader ...
type S3Downloader struct {
	conn *s3.S3
}

type s3ObjectChan struct {
	*s3.Object
	*s3.GetObjectOutput
	err error
}

type s3ObjectListChan struct {
	list []*s3.Object
	err  error
}

// NewS3Downloader creates new S3Downloader
func NewS3Downloader() (*S3Downloader, error) {
	sess, err := session.NewSession()

	if err != nil {
		return nil, err
	}

	return &S3Downloader{
		conn: s3.New(sess),
	}, nil
}

func (s *S3Downloader) makeDownloader(bckt string, filesChan chan s3ObjectChan, count int) chan s3ObjectListChan {
	var groupLock sync.WaitGroup

	listsChan := make(chan s3ObjectListChan, 20)

	bucket := aws.String(bckt)

	downloadChan := make(chan *s3.Object, 100)
	groupLock.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			for obj := range downloadChan {
				res, err := s.conn.GetObject(&s3.GetObjectInput{
					Bucket: bucket,
					Key:    obj.Key,
				})
				filesChan <- s3ObjectChan{obj, res, err}
			}
			groupLock.Done()
		}()
	}

	go func() {
		for elem := range listsChan {
			if elem.err != nil {
				filesChan <- s3ObjectChan{nil, nil, elem.err}
				break
			}
			for _, obj := range elem.list {
				downloadChan <- obj
			}
		}
		close(downloadChan)
		groupLock.Wait()
		close(filesChan)
	}()

	return listsChan
}

// DownloadFileVersion downloads single file with specific version
func (s *S3Downloader) DownloadFileVersion(bucket, key, version string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	return s.conn.GetObject(input)
}

// DownloadFilesByPattern downloads files matching to pattern from bucket.
// It uses 16 goroutines for parallel downloads.
func (s *S3Downloader) DownloadFilesByPattern(bucket, pattern string) (map[string]*s3.GetObjectOutput, error) {
	filesChan := make(chan s3ObjectChan, 100)

	wcParts := strings.SplitN(pattern, "*", 2)

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(wcParts[0]),
		Delimiter: aws.String(""),
	}

	listsChan := s.makeDownloader(bucket, filesChan, 16)

	go func() {
	Loop:
		for {
			output, err := s.conn.ListObjectsV2(input)

			if err != nil {
				listsChan <- s3ObjectListChan{nil, err}
				break
			}

			files := make([]*s3.Object, 0, len(output.Contents))

			for _, obj := range output.Contents {
				r, err := filepath.Match(pattern, *obj.Key)

				if err != nil {
					listsChan <- s3ObjectListChan{nil, err}
					break Loop
				} else if r {
					files = append(files, obj)
				}
			}

			if len(files) > 0 {
				listsChan <- s3ObjectListChan{files, nil}
			}

			if output.NextContinuationToken != nil {
				input.ContinuationToken = output.NextContinuationToken
			} else {
				break
			}
		}
		close(listsChan)
	}()

	res := make(map[string]*s3.GetObjectOutput)

	for obj := range filesChan {
		if obj.err != nil {
			return nil, obj.err
		}
		res[*obj.Key] = obj.GetObjectOutput
	}

	return res, nil
}
