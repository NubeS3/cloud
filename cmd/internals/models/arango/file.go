package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
	"github.com/arangodb/go-driver"
	"io"
	"time"
)

type FileMetadata struct {
	Id       string `json:"id"`
	FileId   string `json:"-"`
	BucketId string `json:"bucket_id"`
	Path     string `json:"path"`
	Name     string `json:"name"`

	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	IsHidden    bool   `json:"is_hidden"`

	IsDeleted   bool      `json:"-"`
	DeletedDate time.Time `json:"-"`

	UploadedDate time.Time `json:"-"`
	ExpiredDate  time.Time `json:"expired_date"`
}

type fileMetadata struct {
	FileId   string `json:"fid"`
	BucketId string `json:"bucket_id"`
	Path     string `json:"path"`
	Name     string `json:"name"`

	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	IsHidden    bool   `json:"is_hidden"`

	IsDeleted   bool      `json:"is_deleted"`
	DeletedDate time.Time `json:"deleted_date"`

	UploadedDate time.Time `json:"upload_date"`
	ExpiredDate  time.Time `json:"expired_date"`
}

func saveFileMetadata(fid string, bid string,
	path string, name string, isHidden bool,
	contentType string, size int64, expiredDate time.Time) (*FileMetadata, error) {
	uploadedTime := time.Time{}

	doc := fileMetadata{
		FileId:       fid,
		BucketId:     bid,
		Path:         path,
		Name:         name,
		ContentType:  contentType,
		Size:         size,
		IsHidden:     isHidden,
		IsDeleted:    false,
		DeletedDate:  time.Time{},
		UploadedDate: uploadedTime,
		ExpiredDate:  expiredDate,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	meta, err := fileMetadataCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return &FileMetadata{
		Id:           meta.Key,
		FileId:       doc.FileId,
		BucketId:     doc.BucketId,
		Path:         doc.Path,
		Name:         doc.Name,
		ContentType:  doc.ContentType,
		Size:         doc.Size,
		IsHidden:     doc.IsHidden,
		IsDeleted:    doc.IsDeleted,
		DeletedDate:  doc.DeletedDate,
		UploadedDate: doc.UploadedDate,
		ExpiredDate:  doc.ExpiredDate,
	}, nil
}

func FindMetadataByFilename(path string, name string, bid string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR fm IN fileMetadata FILTER fm.bucket_id == @bid AND fm.path == @path AND fm.name == @name LIMIT 1 RETURN fm"
	bindVars := map[string]interface{}{
		"bid":  bid,
		"path": path,
		"name": name,
	}

	fm := fileMetadata{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	var retMeta FileMetadata
	for {
		meta, err := cursor.ReadDocument(ctx, &fm)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		retMeta = FileMetadata{
			Id:           meta.Key,
			FileId:       fm.FileId,
			BucketId:     fm.BucketId,
			Path:         fm.Path,
			Name:         fm.Name,
			ContentType:  fm.ContentType,
			Size:         fm.Size,
			IsHidden:     fm.IsHidden,
			IsDeleted:    fm.IsDeleted,
			DeletedDate:  fm.DeletedDate,
			UploadedDate: fm.UploadedDate,
			ExpiredDate:  fm.ExpiredDate,
		}
	}

	return &retMeta, nil
}

func FindMetadataByFid(fid string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR fm IN fileMetadata FILTER fm.fid == @fid LIMIT 1 RETURN fm"
	bindVars := map[string]interface{}{
		"fid": fid,
	}

	fm := fileMetadata{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	var retMeta FileMetadata
	for {
		meta, err := cursor.ReadDocument(ctx, &fm)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		retMeta = FileMetadata{
			Id:           meta.Key,
			FileId:       fm.FileId,
			BucketId:     fm.BucketId,
			Path:         fm.Path,
			Name:         fm.Name,
			ContentType:  fm.ContentType,
			Size:         fm.Size,
			IsHidden:     fm.IsHidden,
			IsDeleted:    fm.IsDeleted,
			DeletedDate:  fm.DeletedDate,
			UploadedDate: fm.UploadedDate,
			ExpiredDate:  fm.ExpiredDate,
		}
	}

	return &retMeta, nil
}

func FindMetadataById(fid string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var data fileMetadata
	meta, err := fileMetadataCol.ReadDocument(ctx, fid, &data)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	if data.IsDeleted || data.ExpiredDate.Before(time.Now()) {
		return nil, &models.ModelError{
			Msg:     "file not found",
			ErrType: models.NotFound,
		}
	}

	return &FileMetadata{
		Id:           meta.Key,
		FileId:       data.FileId,
		BucketId:     data.BucketId,
		Path:         data.Path,
		Name:         data.Name,
		ContentType:  data.ContentType,
		Size:         data.Size,
		IsHidden:     data.IsHidden,
		IsDeleted:    data.IsDeleted,
		DeletedDate:  data.DeletedDate,
		UploadedDate: data.UploadedDate,
		ExpiredDate:  data.ExpiredDate,
	}, nil
}

func SaveFile(reader io.Reader, bid string, bucketName string,
	path string, name string, isHidden bool,
	contentType string, size int64, ttl time.Duration) (*FileMetadata, error) {
	//CHECK BUCKET ID AND NAME
	bucket, err := FindBucketById(bid)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	if bucket.Name != bucketName {
		return nil, &models.ModelError{
			Msg:     "bucket name and id mismatch",
			ErrType: models.InvalidBucket,
		}
	}

	//CHECK PATH

	//CHECK DUP FILE NAME

	meta, err := seaweedfs.UploadFile(bucketName, path, name, size, reader)
	if err != nil {
		return nil, err
	}

	return saveFileMetadata(meta.FileID, bid, path, name, isHidden, contentType, size, time.Now().Add(ttl))
}

func GetFile(bid string, path, name string, callback func(reader io.Reader, metadata *FileMetadata) error) error {
	meta, err := FindMetadataByFilename(path, name, bid)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	//CHECK EXPIRED TIME

	//CHECK FILE DELETE

	err = seaweedfs.DownloadFile(meta.FileId, func(reader io.Reader) error {
		return callback(reader, meta)
	})

	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.FsError,
		}
	}

	return nil
}

func GetFileByFid(fid string, callback func(reader io.Reader, metadata *FileMetadata) error) error {
	fileMeta, err := FindMetadataById(fid)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	err = seaweedfs.DownloadFile(fileMeta.FileId, func(reader io.Reader) error {
		return callback(reader, fileMeta)
	})

	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.FsError,
		}
	}

	return nil
}
