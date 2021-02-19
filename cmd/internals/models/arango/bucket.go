package arango

import (
	"context"
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

func InsertBucket(uid string, name string, region string) (*Bucket, error) {
	createdTime := time.Time{}
	doc := Bucket{
		Uid:       uid,
		Name:      name,
		Region:    region,
		CreatedAt: createdTime,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bucket, _ := FindBucketByName(name)
	if bucket != nil {
		return nil, &ModelError{
			msg:     "duplicated bucket name",
			errType: Duplicated,
		}
	}

	meta, err := bucketCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	doc.Id = meta.Key
	return &doc, nil
}

func FindBucketByName(bname string) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR b IN buckets FILTER b.name == @bname LIMIT 1 RETURN b"
	bindVars := map[string]interface{}{
		"bname": bname,
	}

	bucket := Bucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
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
		return nil, &ModelError{
			msg:     "bucket not found",
			errType: DocumentNotFound,
		}
	}

	return &bucket, nil
}

func FindBucketById(bid string) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bucket := Bucket{}
	meta, err := bucketCol.ReadDocument(ctx, bid, &bucket)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &ModelError{
				msg:     "bucket not found",
				errType: DocumentNotFound,
			}
		}

		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	bucket.Id = meta.Key

	return &bucket, nil
}

func FindBucketByUid(uid string) (*Bucket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR b IN buckets FILTER b.uid == @uid LIMIT 1 RETURN b"
	bindVars := map[string]interface{}{
		"uid": uid,
	}

	bucket := Bucket{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
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
		return nil, &ModelError{
			msg:     "bucket not found",
			errType: DocumentNotFound,
		}
	}

	return &bucket, nil
}

func RemoveBucket(bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := bucketCol.RemoveDocument(ctx, bid)
	if err != nil {
		if driver.IsNotFound(err) {
			return &ModelError{
				msg:     "bucket not found",
				errType: DocumentNotFound,
			}
		}

		return &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	return nil
}
