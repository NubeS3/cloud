package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/arangodb/go-driver"
	"time"
)

type Bucket struct {
	Id     string `json:"id"`
	Uid    string `json:"uid"`
	Name   string `json:"name" binding:"required"`
	Region string `json:"region" binding:"required"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`
}

type bucket struct {
	Uid    string `json:"uid"`
	Name   string `json:"name" binding:"required"`
	Region string `json:"region" binding:"required"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`
}

func InsertBucket(uid string, name string, region string) (*Bucket, error) {
	createdTime := time.Now()
	doc := bucket{
		Uid:       uid,
		Name:      name,
		Region:    region,
		CreatedAt: createdTime,
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
	_ = nats.SendBucketEvent(meta.Key, doc.Uid, doc.Name, doc.Region,
		"Bucket Created", "Add")

	return &Bucket{
		Id:        meta.Key,
		Uid:       doc.Uid,
		Name:      doc.Name,
		Region:    doc.Region,
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
	_ = nats.SendBucketEvent(bucket.Id, bucket.Uid, bucket.Name, bucket.Region,
		"Bucket Delete", "Delete")
	return nil
}
