package models

import (
	"errors"
	"github.com/gocql/gocql"
	"github.com/m1ome/randstr"
	"time"
)

type KeyType int

const (
	Invalid KeyType = iota - 1
	Full
	Write
	Read
)

func (kt KeyType) String() string {
	return [...]string{"Full", "Write", "Read"}[kt]
}

func parseKeyType(t string) KeyType {
	switch t {
	case "Full":
		return Full
	case "Write":
		return Write
	case "Read":
		return Read
	default:
		return Invalid
	}
}

type accessKey struct {
	Key         string
	BucketId    gocql.UUID
	ExpiredDate time.Time
	Type        KeyType
	Uid         gocql.UUID
}

func (a *accessKey) toAccessKey() *AccessKey {
	return &AccessKey{
		Key:         a.Key,
		BucketId:    a.BucketId,
		ExpiredDate: a.ExpiredDate,
		Type:        a.Type.String(),
		Uid:         a.Uid,
	}
}

type AccessKey struct {
	Key         string     `json:"key"`
	BucketId    gocql.UUID `json:"bucket_id"`
	ExpiredDate time.Time  `json:"expired_date"`
	Type        string     `json:"type"`
	Uid         gocql.UUID `json:"uid"`
}

func InsertAccessKey(bId gocql.UUID, uid gocql.UUID,
	sKeyType string, expiredDate time.Time) (*AccessKey, error) {
	key := randstr.GetString(16)
	keyType := parseKeyType(sKeyType)
	if keyType == Invalid {
		return nil, errors.New("invalid key type")
	}

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
		Type:        keyType.String(),
		Uid:         uid,
	}, nil
}

func FindAccessKeyByKey(key string) (*AccessKey, error) {
	var accessKeys []accessKey
	accessKeys = []accessKey{}

	iter := session.
		Query("SELECT FROM access_keys_by_key WHERE key = ? LIMIT 1", key).
		Iter()

	queryAccessKey := accessKey{}
	for iter.Scan(&queryAccessKey.Key, &queryAccessKey.BucketId,
		&queryAccessKey.ExpiredDate, &queryAccessKey.Type, &queryAccessKey.Uid) {
		accessKeys = append(accessKeys, queryAccessKey)
	}

	if len(accessKeys) < 1 {
		return nil, errors.New("access key not found")
	}

	return accessKeys[0].toAccessKey(), nil
}

func FindAccessKeyByUidBid(uid gocql.UUID, bid gocql.UUID) ([]AccessKey, error) {
	var accessKeys []AccessKey
	accessKeys = []AccessKey{}

	iter := session.
		Query("SELECT FROM access_keys_by_uid_bid"+
			" WHERE uid = ? AND bid = ?", uid, bid).
		Iter()

	queryAccessKey := accessKey{}
	for iter.Scan(&queryAccessKey.Key, &queryAccessKey.BucketId,
		&queryAccessKey.ExpiredDate, &queryAccessKey.Type, &queryAccessKey.Uid) {
		accessKeys = append(accessKeys, *queryAccessKey.toAccessKey())
	}

	return accessKeys, nil
}

func DeleteAccessKey(uid gocql.UUID, bid gocql.UUID, key string) error {
	deleteKeyQuery := session.
		Query(`DELETE FROM access_keys_by_key WHERE key = ? AND bucket_id = ?`, key, bid)

	if err := deleteKeyQuery.Exec(); err != nil {
		return err
	}

	deleteKeyUidQuery := session.
		Query(`DELETE FROM access_keys_by_uid_bid WHERE uid = ? AND bucket_id = ? AND key = ?`, uid, bid, key)

	if err := deleteKeyUidQuery.Exec(); err != nil {
		return err
	}

	return nil
}
