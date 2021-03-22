package nats

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"time"
)

type errLogMessage struct {
	Content string    `json:"content"`
	Type    string    `json:"type"`
	At      time.Time `json:"at"`
}

func SendErrorEvent(content, t string) error {
	return c.Publish(errSubj, errLogMessage{
		Content: content,
		Type:    t,
		At:      time.Now(),
	})
}

type fileLog struct {
	Id          string    `json:"id"`
	FId         string    `json:"f_id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	BucketId    string    `json:"bucket_id"`
	ContentType string    `json:"content_type"`
	UploadDate  time.Time `json:"upload_date"`
	Path        string    `json:"path"`
	IsHidden    bool      `json:"is_hidden"`
}

func SendUploadFileEvent(metadata arango.FileMetadata) error {
	return c.Publish(uploadFileSubj, fileLog{
		Id:          metadata.Id,
		FId:         metadata.FileId,
		Name:        metadata.Name,
		Size:        metadata.Size,
		BucketId:    metadata.BucketId,
		ContentType: metadata.ContentType,
		UploadDate:  metadata.UploadedDate,
		Path:        metadata.Path,
		IsHidden:    metadata.IsHidden,
	})
}

func SendDownloadFileEvent(metadata arango.FileMetadata) error {
	return c.Publish(uploadFileSubj, fileLog{
		Id:          metadata.Id,
		FId:         metadata.FileId,
		Name:        metadata.Name,
		Size:        metadata.Size,
		BucketId:    metadata.BucketId,
		ContentType: metadata.ContentType,
		UploadDate:  metadata.UploadedDate,
		Path:        metadata.Path,
		IsHidden:    metadata.IsHidden,
	})
}
