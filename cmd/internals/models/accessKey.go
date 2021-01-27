package models

import (
	"errors"
	"github.com/gocql/gocql"
	"github.com/m1ome/randstr"
	"time"
)

type AccessKey struct {
	Key         string
	BucketId    gocql.UUID
	ExpiredDate time.Time
	Type        string
	Uid         gocql.UUID
}

func InsertAccessKey(bId gocql.UUID, uid gocql.UUID,
	keyType string, expiredDate time.Time) (*AccessKey, error) {
	key := randstr.GetString(16)
	queryTableByKey := session.Query("INSERT INTO access_keys_by_key"+
		" (key, bucket_id, expired_date, type, uid)"+
		" VALUES (?, ?, ?, ?, ?)", key, bId, expiredDate, keyType, uid)

	if err := queryTableByKey.Exec(); err != nil {
		return nil, err
	}

	queryTableByUidBid := session.Query("INSERT INTO access_keys_by_uid_bid"+
		" (key, bucket_id, expired_date, type, uid)"+
		" VALUES (?, ?, ?, ?, ?)", key, bId, expiredDate, keyType, uid)

	if err := queryTableByUidBid.Exec(); err != nil {
		deleteQuery := session.Query("DELETE FROM access_keys_by_key WHERE key = ?", key)
		_ = deleteQuery.Exec()
		return nil, err
	}

	return &AccessKey{
		Key:         key,
		BucketId:    bId,
		ExpiredDate: expiredDate,
		Type:        keyType,
		Uid:         uid,
	}, nil
}

func FindAccessKeyByKey(key string) (*AccessKey, error) {
	var accessKeys []AccessKey
	accessKeys = []AccessKey{}

	iter := session.
		Query("SELECT FROM access_keys_by_key WHERE key = ? LIMIT 1", key).
		Iter()

	queryAccessKey := AccessKey{}
	for iter.Scan(&queryAccessKey.Key, &queryAccessKey.BucketId,
		&queryAccessKey.ExpiredDate, &queryAccessKey.Type, &queryAccessKey.Uid) {
		accessKeys = append(accessKeys, queryAccessKey)
	}

	if len(accessKeys) < 1 {
		return nil, errors.New("access key not found")
	}

	return &accessKeys[0], nil
}
