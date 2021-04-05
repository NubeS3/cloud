package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"time"
)

type keyPair struct {
	Public       string       `json:"public"`
	Private      string       `json:"private"`
	BucketId     string       `json:"bucket_id"`
	GeneratorUid string       `json:"generator_uid"`
	ExpiredDate  time.Time    `json:"expired_date"`
	Permissions  []Permission `json:"permissions"`
}

type KeyPair struct {
	Public       string    `json:"public"`
	Private      string    `json:"private"`
	BucketId     string    `json:"bucket_id"`
	GeneratorUid string    `json:"generator_uid"`
	ExpiredDate  time.Time `json:"expired_date"`
	Permissions  []string  `json:"permissions"`
}

func (k *keyPair) toKeyPair() *KeyPair {
	var perms []string
	for _, perm := range k.Permissions {
		perms = append(perms, perm.String())
	}

	return &KeyPair{
		Public:       k.Public,
		Private:      k.Private,
		BucketId:     k.BucketId,
		GeneratorUid: k.GeneratorUid,
		ExpiredDate:  k.ExpiredDate,
		Permissions:  perms,
	}
}

func GenerateKeyPair(bid, uid string, exp time.Time, perms []string) (*KeyPair, error) {
	public, err := uuid.NewRandom()
	if err != nil {
		return nil, &models.ModelError{
			Msg:     "failed to generate key pair",
			ErrType: models.GeneratorError,
		}
	}

	private, err := uuid.NewRandom()
	if err != nil {
		return nil, &models.ModelError{
			Msg:     "failed to generate key pair",
			ErrType: models.GeneratorError,
		}
	}

	bucket, err := FindBucketById(bid)
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

	doc := keyPair{
		Public:       public.String(),
		Private:      private.String(),
		BucketId:     bid,
		GeneratorUid: uid,
		ExpiredDate:  exp,
		Permissions:  permissions,
	}

	_, err = keyPairsCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return &KeyPair{
		Public:       doc.Public,
		Private:      doc.Private,
		BucketId:     doc.BucketId,
		GeneratorUid: doc.GeneratorUid,
		ExpiredDate:  doc.ExpiredDate,
		Permissions:  perms,
	}, nil
}

func FindKeyByUidBid(uid, bid string, limit, offset int) ([]KeyPair, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR k IN keyPairs FILTER k.bucket_id == @bid AND k.generator_uid == @uid LIMIT @offset, @limit RETURN k"
	bindVars := map[string]interface{}{
		"bid":    bid,
		"uid":    uid,
		"limit":  limit,
		"offset": offset,
	}

	keys := []KeyPair{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		keypair := keyPair{}
		_, err := cursor.ReadDocument(ctx, &keypair)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		keys = append(keys, *keypair.toKeyPair())
	}

	return keys, nil
}

func FindKeyPairByPublic(key string) (*KeyPair, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR kp IN keyPairs FILTER kp.public == @key LIMIT 1 RETURN kp"
	bindVars := map[string]interface{}{
		"key": key,
	}

	kp := keyPair{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &kp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if kp.Public == "" {
		return nil, &models.ModelError{
			Msg:     "key not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return kp.toKeyPair(), nil
}

func RemoveKeyPair(public, bid, uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR kp IN keyPairs FILTER kp.public == @public AND kp.bucket_id == @bid AND kp.generator_uid == @uid REMOVE kp in keyPairs LET removed = OLD RETURN removed"
	bindVars := map[string]interface{}{
		"public": public,
		"bid":    bid,
		"uid":    uid,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	kp := keyPair{}
	for {
		_, err := cursor.ReadDocument(ctx, &kp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if kp.Public == "" {
		return &models.ModelError{
			Msg:     "public not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return nil
}
