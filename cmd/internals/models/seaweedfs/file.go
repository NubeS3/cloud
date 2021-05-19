package seaweedfs

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/linxGnu/goseaweedfs"
	"io"
)

func UploadFile(filename string, size int64, reader io.Reader) (*goseaweedfs.FilePart, error) {

	meta, err := sw.Upload(reader, filename, size, "", "")
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

func DeleteFile(id string) error {
	return sw.DeleteFile(id, nil)
}
