package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"time"
)

type EncryptionInfo struct {
	Id         string     `json:"_,omitempty"`
	BucketId   string     `json:"bucket_id"`
	Passphrase string     `json:"passphrase"`
	From       time.Time  `json:"from"`
	To         *time.Time `json:"to"`
}

func CreateEncrypt(passphrase string, bucketId string) (*EncryptionInfo, error) {
	oldEncrypt, err := FindLatestEncryptionInfoByBucketId(bucketId)
	if err != nil {
		if e, ok := err.(*models.ModelError); ok {
			if e.ErrType != models.NotFound && e.ErrType != models.DocumentNotFound {
				return nil, err
			}
		}
	} else {
		if oldEncrypt.To != nil && oldEncrypt.To.After(time.Now()) {
			return nil, &models.ModelError{
				Msg:     "Encryption Info duplicated",
				ErrType: models.Duplicated,
			}
		}
	}

	from := time.Now()

	doc := EncryptionInfo{
		BucketId:   bucketId,
		Passphrase: passphrase,
		From:       from,
		To:         nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	meta, err := encryptCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	doc.Id = meta.Key

	return &doc, nil
}

func FindLatestEncryptionInfoByBucketId(bucketId string) (*EncryptionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR e IN encryptInfo FILTER e.bucket_id == @bid SORT e.from DESC LIMIT 1 RETURN e"
	bindVars := map[string]interface{}{
		"bid": bucketId,
	}

	encrypt := EncryptionInfo{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		m, err := cursor.ReadDocument(ctx, &encrypt)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		encrypt.Id = m.Key
	}

	if encrypt.Id == "" {
		return nil, &models.ModelError{
			Msg:     "encrypt info not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &encrypt, nil
}

func UpdateToEncryptionInfoByBucketId(bucketId string) (*EncryptionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR e IN encryptInfo FILTER e.bucket_id == @bid AND e.to != nil SORT e.from DESC LIMIT 1 UPDATE e WITH { to: @to } RETURN NEW"
	bindVars := map[string]interface{}{
		"bid": bucketId,
		"to":  time.Now(),
	}

	encrypt := EncryptionInfo{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		m, err := cursor.ReadDocument(ctx, &encrypt)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		encrypt.Id = m.Key
	}

	if encrypt.Id == "" {
		return nil, &models.ModelError{
			Msg:     "encrypt info not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &encrypt, nil
}

func FindEncryptionInfoInDate(bucketId string, date time.Time) (*EncryptionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR e in encryptInfo FILTER e.bucket_id == @bid AND e.from < @d AND (e.to == null OR @d < e.to) LIMIT 1 RETURN e"
	bindVars := map[string]interface{}{
		"bid": bucketId,
		"d":   date,
	}

	encrypt := EncryptionInfo{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		m, err := cursor.ReadDocument(ctx, &encrypt)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		encrypt.Id = m.Key
	}

	if encrypt.Id == "" {
		return nil, &models.ModelError{
			Msg:     "encrypt info not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &encrypt, nil
}
