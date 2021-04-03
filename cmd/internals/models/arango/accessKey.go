package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"github.com/m1ome/randstr"
	"time"
)

type Permission int

const (
	GetFileList Permission = iota
	GetFileListHidden
	Download
	DownloadHidden
	Upload
	MarkHidden
	DeleteFile
	RecoverFile
)

func (perm Permission) String() string {
	return [...]string{
		"GetFileList",
		"GetFileListHidden",
		"Download",
		"DownloadHidden",
		"Upload",
		"MarkHidden",
		"DeleteFile",
		"RecoverFile",
	}[perm]
}

func parsePerm(p string) (Permission, error) {
	switch p {
	case "GetFileList":
		return GetFileList, nil
	case "GetFileListHidden":
		return GetFileListHidden, nil
	case "Download":
		return Download, nil
	case "DownloadHidden":
		return DownloadHidden, nil
	case "Upload":
		return Upload, nil
	case "MarkHidden":
		return MarkHidden, nil
	case "DeleteFile":
		return DeleteFile, nil
	case "RecoverFile":
		return RecoverFile, nil
	default:
		return -1, &models.ModelError{
			Msg:     "invalid permission: " + p,
			ErrType: models.InvalidAccessKey,
		}
	}
}

type accessKey struct {
	Key         string       `json:"key"`
	BucketId    string       `json:"bucket_id"`
	ExpiredDate time.Time    `json:"expired_date"`
	Permissions []Permission `json:"permissions"`
	Uid         string       `json:"uid"`
}

type AccessKey struct {
	Key         string    `json:"key"`
	BucketId    string    `json:"bucket_id"`
	ExpiredDate time.Time `json:"expired_date"`
	Permissions []string  `json:"permissions"`
	Uid         string    `json:"uid"`
}

func (a *accessKey) toAccessKey() *AccessKey {
	var perms []string
	for _, perm := range a.Permissions {
		perms = append(perms, perm.String())
	}

	return &AccessKey{
		Key:         a.Key,
		BucketId:    a.BucketId,
		ExpiredDate: a.ExpiredDate,
		Permissions: perms,
		Uid:         a.Uid,
	}
}

func GenerateAccessKey(bId string, uid string,
	perms []string, expiredDate time.Time) (*AccessKey, error) {
	key := randstr.GetString(16)

	bucket, err := FindBucketById(bId)
	if err != nil {
		return nil, err
	}

	if bucket.Uid != uid {
		return nil, &models.ModelError{
			Msg:     "invalid user",
			ErrType: models.UidMismatch,
		}
	}

	var permissions []Permission
	for _, perm := range perms {
		permission, err := parsePerm(perm)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	doc := accessKey{
		Key:         key,
		BucketId:    bId,
		ExpiredDate: expiredDate,
		Permissions: permissions,
		Uid:         uid,
	}

	_, err = apiKeyCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return &AccessKey{
		Key:         key,
		BucketId:    bId,
		ExpiredDate: expiredDate,
		Permissions: perms,
		Uid:         uid,
	}, nil
}

func FindAccessKeyByKey(key string) (*AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.key == @key LIMIT 1 RETURN k"
	bindVars := map[string]interface{}{
		"key": key,
	}

	akey := accessKey{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if akey.Key == "" {
		return nil, &models.ModelError{
			Msg:     "key not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return akey.toAccessKey(), nil
}

func FindAccessKeyByUidBid(uid string, bid string, limit, offset int) ([]AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.bucket_id == @bid AND k.uid == @uid LIMIT @offset, @limit RETURN k"
	bindVars := map[string]interface{}{
		"bid":    bid,
		"uid":    uid,
		"limit":  limit,
		"offset": offset,
	}

	keys := []AccessKey{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		akey := accessKey{}
		_, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		keys = append(keys, *akey.toAccessKey())
	}

	return keys, nil
}

func DeleteAccessKey(key, bid, uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.key == @key AND k.bucket_id == @bid AND k.uid == @uid REMOVE k in apiKeys LET removed = OLD RETURN removed"
	bindVars := map[string]interface{}{
		"key": key,
		"bid": bid,
		"uid": uid,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	akey := accessKey{}
	for {
		_, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if akey.Key == "" {
		return &models.ModelError{
			Msg:     "key not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return nil
}
