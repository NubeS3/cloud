package models

import (
	"errors"
	"github.com/gocql/gocql"
	"log"
)

type Bucket struct {
	Id     gocql.UUID
	Uid    gocql.UUID
	Name   string
	Region string
}

//func (b *Bucket) Save() error {
//	query := session.
//		Query(`INSERT INTO buckets (id, uid, name, region) VALUES (?, ?, ?, ?) IF NOT EXIST`,
//			b.Id,
//			b.Uid,
//			b.Name,
//			b.Region,
//		)
//	if err := query.Exec(); err != nil {
//		return err
//	}
//	return nil
//}

func InsertBucket(uid gocql.UUID, name string, region string) error {
	id, err := gocql.RandomUUID()
	if err != nil {
		log.Println("gen id failed")
		return err
	}
	query := session.
		Query(`INSERT INTO buckets (id, uid, name, region) VALUES (?, ?, ?, ?) IF NOT EXIST`,
			id,
			uid,
			name,
			region,
		)
	if err := query.Exec(); err != nil {
		return err
	}
	return nil
}

func FindBucketById(uid gocql.UUID, bucketId gocql.UUID) (*Bucket, error) {
	var buckets []Bucket
	buckets = []Bucket{}
	var bucket *Bucket
	iter := session.
		Query(`SELECT * FROM buckets WHERE uid = ? AND id = ?`, uid, bucketId).
		Iter()

	var id gocql.UUID
	var name string
	var region string

	for iter.Scan(&id, &uid, &name, &region) {
		bucket = &Bucket{
			Id:     id,
			Uid:    uid,
			Name:   name,
			Region: region,
		}
		buckets = append(buckets, *bucket)
	}
	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(buckets) < 1 {
		return nil, errors.New("bucket not found")
	}

	return &buckets[0], err
}

func FindBucketByUid(uid gocql.UUID) ([]Bucket, error) {
	var buckets []Bucket
	buckets = []Bucket{}
	var bucket *Bucket
	iter := session.
		Query(`SELECT * FROM buckets WHERE uid = ? AND id = ? LIMIT 1`, uid).
		Iter()

	var id gocql.UUID
	var name string
	var region string

	for iter.Scan(&id, &uid, &name, &region) {
		bucket = &Bucket{
			Id:     id,
			Uid:    uid,
			Name:   name,
			Region: region,
		}
		buckets = append(buckets, *bucket)
	}
	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}
	return buckets, err
}

func Remove(uid gocql.UUID, id gocql.UUID) error {
	query := session.
		Query(`DELETE FROM buckets WHERE uid = ? AND id = ?`, uid, id)

	if err := query.Exec(); err != nil {
		return err
	}
	return nil
}

func UpdateBucketName(id gocql.UUID, newName string) error {
	query := session.
		Query(`UPDATE buckets SET name = ? WHERE id = ?`, newName, id)

	if err := query.Exec(); err != nil {
		return err
	}

	return nil
}
