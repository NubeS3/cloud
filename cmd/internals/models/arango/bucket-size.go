package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"time"
)

type BucketSize struct {
	BucketId string  `json:"bucket_id"`
	Size     float64 `json:"size"`
}

func CreateBucketSize(bucketId string) (*BucketSize, error) {
	doc := BucketSize{
		BucketId: bucketId,
		Size:     0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	_, err := bucketCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return &doc, err
}

func IncreaseBucketSize(bucketId string, size float64) (*BucketSize, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR bs IN bucketSize FILTER bs.bucket_id == @bid UPDATE bs " +
		"WITH { size: bs.size + @size } " +
		"IN bucketSize RETURN NEW"
	bindVars := map[string]interface{}{
		"bid":  bucketId,
		"size": size,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	bucketSize := BucketSize{}
	for {
		_, err := cursor.ReadDocument(ctx, &bucketSize)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	return &bucketSize, err
}

func DecreaseBucketSize(bucketId string, size float64) (*BucketSize, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR bs IN bucketSize FILTER bs.bucket_id == @bid UPDATE bs " +
		"WITH { size: bs.size - @size } " +
		"IN bucketSize RETURN NEW"
	bindVars := map[string]interface{}{
		"bid":  bucketId,
		"size": size,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	bucketSize := BucketSize{}
	for {
		_, err := cursor.ReadDocument(ctx, &bucketSize)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	return &bucketSize, err
}
