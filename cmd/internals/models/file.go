package models

import (
	"github.com/gocql/gocql"
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
