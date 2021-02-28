package seaweedfs

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/linxGnu/goseaweedfs"
	"io"
	"strings"
)

func UploadFile(bucketName string, path string, filename string, size int64, reader io.Reader) (*goseaweedfs.FilePart, error) {
	pathNormalized := strings.ReplaceAll(bucketName+"/"+path+"/"+filename, "/", "_")

	meta, err := sw.Upload(reader, pathNormalized, size, "", "")
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.FsError,
		}
	}

	return meta, nil
}

func DownloadFile(id string, callback func(reader io.Reader) error) error {
	_, err := sw.Download(id, nil, callback)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.FsError,
		}
	}

	return nil
}
