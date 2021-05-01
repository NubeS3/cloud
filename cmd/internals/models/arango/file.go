package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
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
	uploadedTime := time.Now()
	f, err := FindFolderByFullpath(path)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     "folder not found",
			ErrType: models.NotFound,
		}
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	meta, err := fileMetadataCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	_, err = InsertFile(meta.Key, doc.Name, f.Id, doc.ContentType, doc.Size, isHidden)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     "insert file to folder failed",
			ErrType: models.DbError,
		}
	}

	//LOG UPLOAD SUCCESS
	//_ = nats.SendUploadSuccessFileEvent(meta.Key, doc.FileId, doc.Name, doc.Size,
	//	doc.BucketId, doc.ContentType, doc.UploadedDate, doc.Path, doc.IsHidden)

	_ = nats.SendUploadFileEvent(meta.Key, doc.FileId, doc.Name, doc.Size, doc.BucketId, doc.ContentType, doc.UploadedDate, doc.Path, doc.IsHidden)
	_, err = IncreaseBucketSize(doc.BucketId, float64(doc.Size))
	if err != nil {
		return nil, &models.ModelError{
			Msg:     "failed to increase bucket size, " + err.Error(),
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

func FindMetadataByBid(bid string, limit int64, offset int64, showHidden bool) ([]FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	var query string
	if showHidden {
		query = "FOR fm IN fileMetadata FILTER fm.bucket_id == @bid AND fm.is_deleted != true " +
			"LIMIT @offset, @limit RETURN fm"
	} else {
		query = "FOR fm IN fileMetadata FILTER fm.bucket_id == @bid AND fm.is_deleted != true " +
			"AND fm.is_hidden == false LIMIT @offset, @limit RETURN fm"
	}

	bindVars := map[string]interface{}{
		"bid":    bid,
		"offset": offset,
		"limit":  limit,
	}

	fileMetadatas := []FileMetadata{}
	fileMetadata := fileMetadata{}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &fileMetadata)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		if !fileMetadata.IsDeleted {
			fileMetadatas = append(fileMetadatas, FileMetadata{
				Id:           meta.Key,
				FileId:       fileMetadata.FileId,
				BucketId:     fileMetadata.BucketId,
				Path:         fileMetadata.Path,
				Name:         fileMetadata.Name,
				ContentType:  fileMetadata.ContentType,
				Size:         fileMetadata.Size,
				IsHidden:     fileMetadata.IsHidden,
				IsDeleted:    fileMetadata.IsDeleted,
				DeletedDate:  fileMetadata.DeletedDate,
				UploadedDate: fileMetadata.UploadedDate,
				ExpiredDate:  fileMetadata.ExpiredDate,
			})
		}
	}

	return fileMetadatas, nil
}

func FindMetadataByFilename(path string, name string, bid string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fm IN fileMetadata FILTER fm.bucket_id == @bid AND fm.path == @path AND fm.name == @name AND fm.is_deleted != true LIMIT 1 RETURN fm"
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

	if retMeta.Id == "" || retMeta.IsDeleted || retMeta.ExpiredDate.Before(time.Now()) {
		return nil, &models.ModelError{
			Msg:     "not found",
			ErrType: models.NotFound,
		}
	}

	return &retMeta, nil
}

func FindMetadataByFid(fid string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fm IN fileMetadata FILTER fm.fid == @fid AND fm.is_deleted != true  LIMIT 1 RETURN fm"
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

	if retMeta.IsDeleted || retMeta.ExpiredDate.Before(time.Now()) {
		return nil, &models.ModelError{
			Msg:     "file not found",
			ErrType: models.NotFound,
		}
	}

	return &retMeta, nil
}

func FindMetadataById(fid string) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
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

func SaveFile(reader io.Reader, bid string,
	path string, name string, isHidden bool,
	contentType string, size int64, ttl time.Duration) (*FileMetadata, error) {
	//CHECK BUCKET ID AND NAME
	_, err := FindBucketById(bid)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.NotFound {
				return nil, err
			}
		}
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	if ttl == time.Duration(0) {
		ttl = time.Hour * 24 * 365 * 10
	}

	//CHECK DUP FILE NAME
	_, err = FindMetadataByFilename(path, name, bid)
	if err == nil {
		return nil, &models.ModelError{
			Msg:     "duplicate file",
			ErrType: models.Duplicated,
		}
	}

	//LOG STAGING
	//_ = nats.SendStagingFileEvent(name, size, bid, contentType, path, isHidden)

	meta, err := seaweedfs.UploadFile(name, size, reader)
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

func GetFileByFidIgnoreQueryMetadata(fid string, callback func(reader io.Reader) error) error {
	err := seaweedfs.DownloadFile(fid, callback)

	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.FsError,
		}
	}

	return nil
}

func ToggleHidden(fullpath string, isHidden bool) (*FileMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR fm IN fileMetadata FILTER fm.path == @fullpath UPDATE fm WITH { is_hidden: @isHidden} IN fileMetadata RETURN NEW"
	bindVars := map[string]interface{}{
		"fullpath": fullpath,
		"isHidden": isHidden,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	fileMetadata := FileMetadata{}
	for {
		meta, err := cursor.ReadDocument(ctx, &fileMetadata)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		fileMetadata.Id = meta.Key
	}

	if fileMetadata.Id == "" {
		return nil, &models.ModelError{
			Msg:     "folder not found",
			ErrType: models.DocumentNotFound,
		}
	}

	_, err = UpdateHiddenStatusOfFolderChild(fileMetadata.Path, fileMetadata.Id,
		fileMetadata.Name, fileMetadata.IsHidden)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return &fileMetadata, nil
}

func MarkDeleteFile(path string, name string, bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	deleteDate := time.Now()
	query := "FOR fm IN fileMetadata FILTER fm.bucket_id == @bid AND fm.path == @path AND fm.name == @name LIMIT 1 " +
		"UPDATE fm " +
		"WITH { is_deleted: true, deleted_date: @del_date } " +
		"IN fileMetadata RETURN NEW"
	bindVars := map[string]interface{}{
		"bid":      bid,
		"path":     path,
		"name":     name,
		"del_date": deleteDate,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	fm := FileMetadata{}
	for {
		meta, err := cursor.ReadDocument(ctx, &fm)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		fm.Id = meta.Key
	}

	if fm.Id == "" {
		return &models.ModelError{
			Msg:     "file not found",
			ErrType: models.DocumentNotFound,
		}
	}

	_, err = RemoveChildOfFolderByPath(fm.Path, Child{
		Id:       fm.Id,
		Name:     fm.Name,
		Type:     "file",
		IsHidden: fm.IsHidden,
		Metadata: ChildFileMetadata{
			ContentType: fm.ContentType,
			Size:        fm.Size,
		},
	})
	if err != nil {
		return err
	}

	_, err = DecreaseBucketSize(fm.BucketId, float64(fm.Size))
	if err != nil {
		return err
	}

	return nil
}
