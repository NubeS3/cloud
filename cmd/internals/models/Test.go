package models

import (
	"fmt"
	"github.com/gocql/gocql"
	"github.com/linxGnu/goseaweedfs"
	"io"
	"log"
	"mime/multipart"
)

func TestDb() string {
	var id gocql.UUID
	var name string

	iter := session.
		Query(`SELECT id, name FROM test WHERE name = ? LIMIT 1`, "Rin").
		Consistency(gocql.One).
		Iter()

	for iter.Scan(&id, &name) {
		fmt.Println("DATA:", id, name)
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}

	return name
}

func TestUpload(fileContent multipart.File, size int64, newPath string, collection string, ttl string) (*goseaweedfs.FilerUploadResult, error) {
	filers := sw.Filers()
	filer := filers[0]
	res, err := filer.Upload(fileContent, size, newPath, collection, ttl)
	if err != nil {
		return res, err
	}
	return res, nil
}

func TestDelete(path string) error {
	filers := sw.Filers()
	filer := filers[0]
	err := filer.Delete(path, nil)
	return err
}

func TestDownload(path string, callback func(r io.Reader) error) error {
	filers := sw.Filers()
	filer := filers[0]
	err := filer.Download(path, nil, callback)
	return err
}
