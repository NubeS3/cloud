package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"time"
)

type Bucket struct {
	Id   string `json:"id"`
	Uid  string `json:"uid"`
	Name string `json:"name" binding:"required"`
	//Region string `json:"region" binding:"required"`

	IsPublic     bool `json:"is_public"`
	IsEncrypted  bool `json:"is_encrypted"`
	IsObjectLock bool `json:"is_object_lock"`

	// DB Info
	CreatedAt time.Time `json:"created_at"`

	HoldDuration time.Duration `json:"hold_duration"`
}

type bucket struct {
	Uid          string `json:"uid"`
	Name         string `json:"name"`
	IsPublic     bool   `json:"is_public"`
	IsEncrypted  bool   `json:"is_encrypted"`
	IsObjectLock bool   `json:"is_object_lock"`

	//Region string `json:"region" binding:"required"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`

	HoldDuration time.Duration `json:"hold_duration"`
}

type DetailBucket struct {
	Bucket      Bucket  `json:"bucket"`
	Size        float64 `json:"size"`
	ObjectCount int64   `json:"object_count"`
}

func InsertBucket(uid string, name string, isPublic, isEncrypted, isObjectLock bool) (*Bucket, error) {
	createdTime := time.Now()
	doc := bucket{
		Uid:          uid,
		Name:         name,
		IsPublic:     isPublic,
		IsEncrypted:  isEncrypted,
		IsObjectLock: isObjectLock,
		//Region:    region,
		CreatedAt:    createdTime,
		HoldDuration: 0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	bucket, _ := FindBucketByName(name)
	if bucket != nil {
		return nil, &models.ModelError{
			Msg:     "duplicated bucket name",
			ErrType: models.Duplicated,
		}
	}

	meta, err := bucketCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	_, err = InsertBucketFolder(doc.Name)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	_, err = CreateBucketSize(meta.Key)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	//LOG CREATE BUCKET
	//_ = nats.SendBucketEvent(meta.Key, doc.Uid, doc.Name, doc.Region, "create")

	return &Bucket{
		Id:           meta.Key,
		Uid:          doc.Uid,
		Name:         doc.Name,
		IsPublic:     doc.IsPublic,
		IsEncrypted:  doc.IsEncrypted,
		IsObjectLock: doc.IsObjectLock,

		CreatedAt: doc.CreatedAt,
	}, nil
}

func FindBucketByName(bname string) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR b IN buckets FILTER b.name == @bname LIMIT 1 RETURN b"
	bindVars := map[string]interface{}{
		"bname": bname,
	}

	bucket := Bucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Id = meta.Key
	}

	if bucket.Id == "" {
		return nil, &models.ModelError{
			Msg:     "bucket not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &bucket, nil
}

func FindDetailBucketByName(bname string) (*DetailBucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR b IN buckets FILTER b.name == @bname LIMIT 1" +
		" let size =" +
		" (for s in bucketSize filter s.bucket_id == b._key limit 1 return s.size)" +
		" let objectCount =" +
		" (for fm in fileMetadata filter fm.is_deleted != false and fm.bucket_id == b._key" +
		" collect with count into c return c) " +
		" RETURN {_key: b._key, bucket: b, size: FIRST(size), object}"
	bindVars := map[string]interface{}{
		"bname": bname,
	}

	bucket := DetailBucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Bucket.Id = meta.Key
	}

	if bucket.Bucket.Id == "" {
		return nil, &models.ModelError{
			Msg:     "bucket not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &bucket, nil
}

func FindBucketById(bid string) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	bucket := Bucket{}
	meta, err := bucketCol.ReadDocument(ctx, bid, &bucket)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &models.ModelError{
				Msg:     "bucket not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	bucket.Id = meta.Key

	return &bucket, nil
}

func FindDetailBucketById(bid string) (*DetailBucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR b IN buckets FILTER b._key == @bid LIMIT 1" +
		" let size =" +
		" (for s in bucketSize filter s.bucket_id == b._key limit 1 return s.size)" +
		" let objectCount =" +
		" (for fm in fileMetadata filter fm.is_deleted != false and fm.bucket_id == b._key" +
		" collect with count into c return c) " +
		" RETURN {_key: b._key, bucket: b, size: FIRST(size), object}"
	bindVars := map[string]interface{}{
		"bid": bid,
	}

	bucket := DetailBucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Bucket.Id = meta.Key
	}

	if bucket.Bucket.Id == "" {
		return nil, &models.ModelError{
			Msg:     "bucket not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &bucket, nil
}

func FindAllBucket(limit int64, offset int64) ([]Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR b IN buckets LIMIT @offset, @limit RETURN b"
	bindVars := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	buckets := []Bucket{}
	bucket := Bucket{}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Id = meta.Key
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

func FindBucketByUid(uid string, limit int64, offset int64) ([]Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR b IN buckets FILTER b.uid == @uid LIMIT @offset, @limit RETURN b"
	bindVars := map[string]interface{}{
		"uid":    uid,
		"limit":  limit,
		"offset": offset,
	}

	buckets := []Bucket{}
	bucket := Bucket{}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Id = meta.Key
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

func FindDetailBucketByUid(uid string, limit int64, offset int64) ([]DetailBucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "for b in buckets" +
		" filter b.uid == @uid" +
		" limit @offset, @limit" +
		" let size =" +
		" (for s in bucketSize filter s.bucket_id == b._key limit 1 return s.size)" +
		" let objectCount =" +
		" (for fm in fileMetadata filter fm.is_deleted != false and fm.bucket_id == b._key" +
		" collect with count into c return c) " +
		" return {_key: b._key, bucket: b, size: FIRST(size), object_count: FIRST(objectCount)}"
	bindVars := map[string]interface{}{
		"uid":    uid,
		"limit":  limit,
		"offset": offset,
	}

	buckets := []DetailBucket{}
	bucket := DetailBucket{}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Bucket.Id = meta.Key
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

func UpdateBucketById(bid string, isPublic, isEncrypted, isObjectLock *bool) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	updateTarget := "{ "
	addComa := false
	bindVars := make(map[string]interface{})
	bindVars["id"] = bid

	if isPublic != nil {
		updateTarget += "is_public: @isPublic"
		addComa = true
		bindVars["isPublic"] = isPublic
	}
	if isEncrypted != nil {
		if addComa {
			updateTarget += ", "
		}

		updateTarget += "is_encrypted: @isEncrypted"
		bindVars["isEncrypted"] = isEncrypted
	}
	if isObjectLock != nil {
		if addComa {
			updateTarget += ", "
		}

		updateTarget += "is_object_lock: @isLock"
		bindVars["isLock"] = isObjectLock
	}
	updateTarget += " }"

	query := "FOR b IN buckets FILTER b._key == @id " +
		"UPDATE b WITH " + updateTarget + " IN buckets RETURN NEW"
	//bindVars := map[string]interface{}{
	//	"id":          bid,
	//	"isPublic":    isPublic,
	//	"isEncrypted": isEncrypted,
	//	"isLock":      isObjectLock,
	//}

	bucket := Bucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		bucket.Id = meta.Key
	}

	if bucket.Id == "" {
		return nil, &models.ModelError{
			Msg:     "bucket not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &bucket, nil
}

func UpdateHoldDuration(bid string, duration time.Duration) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR b IN buckets FILTER b._key == @id " +
		"UPDATE b WITH { hold_duration: @duration } IN buckets RETURN NEW"
	bindVars := map[string]interface{}{
		"id":       bid,
		"duration": duration,
	}

	bucket := Bucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &bucket)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		bucket.Id = meta.Key
	}

	if bucket.Id == "" {
		return nil, &models.ModelError{
			Msg:     "bucket not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &bucket, nil
}

func RemoveBucket(uid string, bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	bucket, err := FindBucketById(bid)
	if err != nil {
		return &models.ModelError{
			Msg:     "bucket not found",
			ErrType: models.DocumentNotFound,
		}
	}
	if bucket.Uid != uid {
		return &models.ModelError{
			Msg:     "uid mismatch",
			ErrType: models.UidMismatch,
		}
	}

	err = RemoveFolderAndItsChildren("", bucket.Name)
	if err != nil {
		return err
	}

	_, err = bucketCol.RemoveDocument(ctx, bid)
	if err != nil {
		if driver.IsNotFound(err) {
			return &models.ModelError{
				Msg:     "bucket not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	//LOG CREATE BUCKET
	//_ = nats.SendBucketEvent(bucket.Id, bucket.Uid, bucket.Name, bucket.Region, "delete")

	return nil
}
