package models

import (
	"github.com/gocql/gocql"
	"github.com/linxGnu/goseaweedfs"
	"mime/multipart"
	"time"
)

type FileMetadata struct {
	Id       gocql.UUID
	BucketId gocql.UUID
	Parent   string
	Name     string

	ContentType string
	Size        int64
	IsHidden    bool

	IsDeleted   bool
	DeletedDate time.Time

	UploadedDate time.Time
	ExpiredDate  time.Time
}

func UploadFile(fileContent multipart.File, size int64, newPath string, collection string, ttl string) (*goseaweedfs.FilerUploadResult, error) {
	filers := sw.Filers()
	filer := filers[0]
	res, err := filer.Upload(fileContent, size, newPath, collection, ttl)
	if err != nil {
		return res, err
	}
	return res, nil
}
