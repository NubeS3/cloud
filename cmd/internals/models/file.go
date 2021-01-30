package models

import (
	"fmt"
	"github.com/gocql/gocql"
	"io"
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
		"is_deleted, deleted_date, upload_date, expired_date)"+
		" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, bid, path, name, contentType, size, isHidden, false, time.Time{}, uploadedDate, expiredDate)

	if err := queryBid.Exec(); err != nil {
		return nil, err
	}

	queryId := session.Query("INSERT INTO file_metadata_by_id "+
		"(id, bucket_id, path, name, content_type, size, is_hidden, "+
		"is_deleted, deleted_date, upload_date, expired_date)"+
		" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
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

	if metadata.IsDeleted {
		return nil
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

func MarkDeletedFileMetadata(bucketId gocql.UUID, id gocql.UUID) error {
	deletedDate := time.Now()

	query := session.
		Query("UPDATE file_metadata_by_id"+
			" SET is_deleted = ? AND deleted_date = ?"+
			" WHERE bucket_id = ? AND id = ?", true, deletedDate, bucketId, id)

	if err := query.Exec(); err != nil {
		return err
	}

	return nil
}

func SaveFile(reader io.Reader, bid gocql.UUID, bucketName string,
	path string, name string, isHidden bool,
	contentType string, size int64, ttl time.Duration) (*FileMetadata, error) {
	f, err := swFiler.Upload(reader, size, bucketName+path+name, bucketName, "")
	if err != nil {
		return nil, err
	}

	fmt.Println(f.FileID)
	fmt.Println(bucketName + path + name)

	return InsertFileMetadata(bid, path, name, isHidden, contentType, size, time.Now().Add(ttl))
}
