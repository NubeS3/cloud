package models

import (
	"fmt"
	"github.com/gocql/gocql"
	"log"
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
