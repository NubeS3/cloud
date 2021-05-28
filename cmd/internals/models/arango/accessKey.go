package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/arangodb/go-driver"
	"github.com/m1ome/randstr"
	"time"
)

type Permission int

const (
	ListKeys Permission = iota
	WriteKey
	DeleteKey
	ListBuckets
	WriteBucket
	DeleteBucket
	ReadBucketEncryption
	WriteBucketEncryption
	ReadBucketRetentions
	WriteBucketRetentions
	ListFiles
	ReadFiles
	ShareFiles
	WriteFiles
	DeleteFiles
	LockFiles
	NA
)

func (perm Permission) String() string {
	return [...]string{
		"ListKeys",
		"WriteKey",
		"DeleteKey",
		"ListBuckets",
		"WriteBucket",
		"DeleteBucket",
		"ReadBucketEncryption",
		"WriteBucketEncryption",
		"ReadBucketRetentions",
		"WriteBucketRetentions",
		"ListFiles",
		"ReadFiles",
		"ShareFiles",
		"WriteFiles",
		"DeleteFiles",
		"LockFiles",
	}[perm]
}

func ParsePerm(p string) (Permission, error) {
	switch p {
	case "ListKeys":
		return ListKeys, nil
	case "WriteKey":
		return WriteKey, nil
	case "DeleteKey":
		return DeleteKey, nil
	case "ListBuckets":
		return ListBuckets, nil
	case "WriteBucket":
		return WriteBucket, nil
	case "DeleteBucket":
		return DeleteBucket, nil
	case "ReadBucketEncryption":
		return ReadBucketEncryption, nil
	case "WriteBucketEncryption":
		return WriteBucketEncryption, nil
	case "ReadBucketRetentions":
		return ListFiles, nil
	case "WriteBucketRetentions":
		return ListFiles, nil
	case "ListFiles":
		return ListFiles, nil
	case "ReadFiles":
		return ReadFiles, nil
	case "ShareFiles":
		return ShareFiles, nil
	case "WriteFiles":
		return WriteFiles, nil
	case "DeleteFiles":
		return DeleteFiles, nil
	case "LockFiles":
		return LockFiles, nil
	default:
		return NA, &models.ModelError{
			Msg:     "Invalid Permission",
			ErrType: models.InvalidAccessKey,
		}
	}
}

const (
	MASTER_KEY_TYPE = "MASTER"
	APP_KEY         = "APP"
)

type accessKey struct {
	Name                   string       `json:"name"`
	Key                    string       `json:"key"`
	BucketId               string       `json:"bucket_id"`
	ExpiredDate            time.Time    `json:"expired_date"`
	FileNamePrefixRestrict string       `json:"file_name_prefix_restrict"`
	Permissions            []Permission `json:"permissions"`
	Uid                    string       `json:"uid"`
	KeyType                string       `json:"type"`
}

type AccessKey struct {
	Id                     string    `json:"id"`
	Name                   string    `json:"name"`
	Key                    string    `json:"key"`
	BucketId               string    `json:"bucket_id"`
	ExpiredDate            time.Time `json:"expired_date"`
	FileNamePrefixRestrict string    `json:"file_name_prefix_restrict"`
	Permissions            []string  `json:"permissions"`
	Uid                    string    `json:"uid"`
	KeyType                string    `json:"type"`
}

func (a *accessKey) toAccessKey(id string) *AccessKey {
	var perms []string
	for _, perm := range a.Permissions {
		perms = append(perms, perm.String())
	}

	return &AccessKey{
		Id:                     id,
		Name:                   a.Name,
		Key:                    a.Key,
		BucketId:               a.BucketId,
		ExpiredDate:            a.ExpiredDate,
		Permissions:            perms,
		Uid:                    a.Uid,
		KeyType:                a.KeyType,
		FileNamePrefixRestrict: a.FileNamePrefixRestrict,
	}
}

func GenerateApplicationKey(name string, bid *string, uid string,
	perms []string, expiredDate time.Time, filenamePrefix string) (*AccessKey, error) {
	key := randstr.GetString(16)

	var targetBucketId string
	if bid != nil {
		_, err := FindBucketById(*bid)
		if err != nil {
			return nil, err
		}

		targetBucketId = *bid
	} else {
		targetBucketId = "*"
	}

	var permissions []Permission
	for _, perm := range perms {
		permission, err := ParsePerm(perm)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	doc := accessKey{
		Name:                   name,
		Key:                    key,
		BucketId:               targetBucketId,
		ExpiredDate:            expiredDate,
		Permissions:            permissions,
		Uid:                    uid,
		KeyType:                APP_KEY,
		FileNamePrefixRestrict: filenamePrefix,
	}

	meta, err := apiKeyCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	//LOG CREATE ACCESS KEY
	//var perm []string
	//for _, p := range doc.Permissions {
	//	perm = append(perm, p.String())
	//}
	//_ = nats.SendAccessKeyEvent(doc.Key, doc.BucketId, doc.ExpiredDate,
	//	perm, doc.Uid, "create")

	return doc.toAccessKey(meta.Key), nil
}

func GenerateMasterKey(uid string) (*AccessKey, error) {
	key := randstr.GetString(16)

	user, err := FindUserById(uid)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	doc := accessKey{
		Name:                   "Master Key",
		Key:                    key,
		BucketId:               "*",
		ExpiredDate:            time.Now().Add(time.Hour * 24 * 365 * 100),
		FileNamePrefixRestrict: "",
		Permissions: []Permission{
			ListKeys,
			WriteKey,
			DeleteKey,
			ListBuckets,
			WriteBucket,
			DeleteBucket,
			ReadBucketEncryption,
			WriteBucketEncryption,
			ReadBucketRetentions,
			WriteBucketRetentions,
			ListFiles,
			ReadFiles,
			ShareFiles,
			WriteFiles,
			DeleteFiles,
			LockFiles,
		},
		Uid:     user.Id,
		KeyType: MASTER_KEY_TYPE,
	}

	meta, err := apiKeyCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return doc.toAccessKey(meta.Key), nil
}

func FindMasterKeyByUid(uid string) (*AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.uid == @uid AND k.type == @t LIMIT 1 RETURN k"
	bindVars := map[string]interface{}{
		"uid": uid,
		"t":   "MASTER",
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

	var meta driver.DocumentMeta
	for {
		m, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		meta = m
	}

	if akey.Key == "" {
		return nil, &models.ModelError{
			Msg:     "key not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return akey.toAccessKey(meta.Key), nil
}

func FindAppKeyByNameAndUid(uid, name string) (*AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.uid == @uid AND k.name == @name AND k.type == @t LIMIT 1 RETURN k"
	bindVars := map[string]interface{}{
		"uid":  uid,
		"name": name,
		"t":    "APP",
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

	var meta driver.DocumentMeta
	for {
		m, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		meta = m
	}

	if akey.Key == "" {
		return nil, &models.ModelError{
			Msg:     "key not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return akey.toAccessKey(meta.Key), nil
}

func FindAccessKeyById(id string) (*AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var key accessKey
	meta, err := apiKeyCol.ReadDocument(ctx, id, &key)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return key.toAccessKey(meta.Key), err
}

func GetAccessKeyByUid(uid string, limit, offset int) ([]AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.uid == @uid LIMIT @offset, @limit RETURN k"
	bindVars := map[string]interface{}{
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
		meta, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		akey.Key = "*****"
		keys = append(keys, *akey.toAccessKey(meta.Key))
	}

	return keys, nil
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

	var meta driver.DocumentMeta
	for {
		meta, err = cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if meta.Key == "" {
		return nil, &models.ModelError{
			Msg:     "key not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return akey.toAccessKey(meta.Key), nil
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
		meta, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		keys = append(keys, *akey.toAccessKey(meta.Key))
	}

	return keys, nil
}

func FindAccessKeyByBid(bid string, limit, offset int) ([]AccessKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR k IN apiKeys FILTER k.bucket_id == @bid LIMIT @offset, @limit RETURN k"
	bindVars := map[string]interface{}{
		"bid":    bid,
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
		meta, err := cursor.ReadDocument(ctx, &akey)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		keys = append(keys, *akey.toAccessKey(meta.Key))
	}

	return keys, nil
}

func DeleteAccessKeyById(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := apiKeyCol.RemoveDocument(ctx, key)
	return err
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

	//LOG DELETE ACCESS KEY
	var perm []string
	for _, p := range akey.Permissions {
		perm = append(perm, p.String())
	}
	_ = nats.SendAccessKeyEvent(akey.Key, akey.BucketId, akey.Uid, "Delete")

	return nil
}
