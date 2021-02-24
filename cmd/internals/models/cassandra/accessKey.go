package cassandra

import (
	"errors"
	"github.com/gocql/gocql"
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
	UploadHidden
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
		"UploadHidden",
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
	case "UploadHidden":
		return UploadHidden, nil
	case "DeleteFile":
		return DeleteFile, nil
	case "RecoverFile":
		return RecoverFile, nil
	default:
		return -1, errors.New("invalid permission type")
	}
}

type accessKey struct {
	Key         string
	BucketId    gocql.UUID
	ExpiredDate time.Time
	Permissions []Permission
	Uid         gocql.UUID
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

type AccessKey struct {
	Key         string     `json:"key"`
	BucketId    gocql.UUID `json:"bucket_id"`
	ExpiredDate time.Time  `json:"expired_date"`
	Permissions []string   `json:"permissions"`
	Uid         gocql.UUID `json:"uid"`
}

func InsertAccessKey(bId gocql.UUID, uid gocql.UUID,
	perms []string, expiredDate time.Time) (*AccessKey, error) {
	key := randstr.GetString(16)

	var permissions []Permission
	for _, perm := range perms {
		permission, err := parsePerm(perm)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	queryTableByKey := session.Query("INSERT INTO access_keys_by_key"+
		" (key, bucket_id, expired_date, permissions, uid)"+
		" VALUES (?, ?, ?, ?, ?)", key, bId, expiredDate, permissions, uid)

	if err := queryTableByKey.Exec(); err != nil {
		return nil, err
	}

	queryTableByUidBid := session.Query("INSERT INTO access_keys_by_uid_bid"+
		" (key, bucket_id, expired_date, permissions, uid)"+
		" VALUES (?, ?, ?, ?, ?)", key, bId, expiredDate, permissions, uid)

	if err := queryTableByUidBid.Exec(); err != nil {
		deleteQuery := session.Query("DELETE FROM access_keys_by_key WHERE key = ?", key)
		_ = deleteQuery.Exec()
		return nil, err
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
	var accessKeys []accessKey
	accessKeys = []accessKey{}

	iter := session.
		Query("SELECT * FROM access_keys_by_key WHERE key = ? LIMIT 1", key).
		Iter()

	queryAccessKey := accessKey{}
	for iter.Scan(&queryAccessKey.Key, &queryAccessKey.BucketId,
		&queryAccessKey.ExpiredDate, &queryAccessKey.Permissions, &queryAccessKey.Uid) {
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
		&queryAccessKey.ExpiredDate, &queryAccessKey.Permissions, &queryAccessKey.Uid) {
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
