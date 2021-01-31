package models

import (
	"github.com/gocql/gocql"
	"time"
)

type FileMetadata struct {
	Id       gocql.UUID
	BucketId gocql.UUID
	Path     string
	Name     string

	ContentType string
	Size        int64
	IsHidden    bool

	IsDeleted   bool
	DeletedDate time.Time

	UploadedDate time.Time
	ExpiredDate  time.Time
}

func InsertFileMetadata(bid gocql.UUID,
	path string, name string, isHidden bool,
	contentType string, size int64, expiredDate time.Time) (*FileMetadata, error) {
	id, err := gocql.RandomUUID()
	if err != nil {
		return nil, err
	}

	uploadedDate := time.Now()

	queryBid := session.Query("INSERT INTO file_metadata_by_bid "+
		"(id, bucket_id, path, name, content_type, size, is_hidden, "+
		"is_deleted, deleted_date, upload_date, expired_date)",
		id, bid, path, name, contentType, size, isHidden, false, time.Time{}, uploadedDate, expiredDate)

	if err := queryBid.Exec(); err != nil {
		return nil, err
	}

	queryId := session.Query("INSERT INTO file_metadata_by_id "+
		"(id, bucket_id, path, name, content_type, size, is_hidden, "+
		"is_deleted, deleted_date, upload_date, expired_date)",
		id, bid, path, name, contentType, size, isHidden, false, time.Time{}, uploadedDate, expiredDate)

	if err := queryId.Exec(); err != nil {
		deleteQuery := session.Query("DELETE FROM file_metadata_by_bid"+
			" WHERE bid = ? AND upload_date = ? AND id = ?", bid, uploadedDate, id)
		_ = deleteQuery.Exec()
		return nil, err
	}

	return &FileMetadata{
		Id:           id,
		BucketId:     bid,
		Path:         path,
		Name:         name,
		ContentType:  contentType,
		Size:         size,
		IsHidden:     false,
		IsDeleted:    false,
		DeletedDate:  time.Time{},
		UploadedDate: uploadedDate,
		ExpiredDate:  time.Time{},
	}, nil
}

func GetFileMetadataById(bucketId gocql.UUID, id gocql.UUID) *FileMetadata {
	iter := session.
		Query("SELECT FROM file_metadata_by_id"+
			" WHERE bucket_id = ? AND id = ? LIMIT 1", bucketId, id).
		Iter()

	metadata := FileMetadata{}
	for iter.Scan(&metadata.Id, &metadata.BucketId, &metadata.Path, &metadata.Name,
		&metadata.ContentType, &metadata.Size, &metadata.IsHidden, &metadata.IsDeleted,
		&metadata.DeletedDate, &metadata.UploadedDate, &metadata.ExpiredDate) {

	}

	return &metadata
}

func GetFileMetadataByBucketId(bucketId gocql.UUID) []FileMetadata {
	iter := session.
		Query("SELECT FROM file_metadata_by_id"+
			" WHERE bucket_id = ?", bucketId).
		Iter()

	var metadata []FileMetadata
	var metadatum FileMetadata
	for iter.Scan(&metadatum.Id, &metadatum.BucketId, &metadatum.Path, &metadatum.Name,
		&metadatum.ContentType, &metadatum.Size, &metadatum.IsHidden, &metadatum.IsDeleted,
		&metadatum.DeletedDate, &metadatum.UploadedDate, &metadatum.ExpiredDate) {
		metadata = append(metadata, metadatum)
	}

	return metadata
}
