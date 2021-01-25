package models

import (
	"github.com/gocql/gocql"
	"log"
	"time"
)

type User struct {
	Id           gocql.UUID
	Firstname    string
	Lastname     string
	Username     string
	Pass         string
	Email        string
	Dob          time.Time
	Company      string
	Gender       bool
	RefreshToken string
	ExpiredRf    time.Time
	IsBanned     bool
	// DB Info
	CreatedAt time.Time
	UpdatedAt time.Time
}

func FindUserById(id gocql.UUID) (*User, error) {
	iter := session.
		Query(`SELECT * FROM userbyuid WHERE username = ?`, id).
		Consistency(gocql.One).
		Iter()
	var username string
	var email string
	var dob time.Time
	var pass string
	var company string
	var gender bool
	for iter.Scan(&id, &company, &dob, &email, &gender, &pass, &username) {
		println("Scanned")
	}
	user := &User{
		Id:       id,
		Username: username,
		Email:    email,
		Dob:      dob,
		Pass:     pass,
		Company:  company,
		Gender:   gender,
	}
	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}
	return user, err
}

func FindUserByUsername(username string) (*User, error) {
	iter := session.
		Query(`SELECT * FROM userbyuid WHERE username = ? `, username).
		Consistency(gocql.One).
		Iter()
	var id gocql.UUID
	var email string
	var dob time.Time
	var pass string
	var company string
	var gender bool
	for iter.Scan(&id, &company, &dob, &email, &gender, &pass, &username) {
		println("DATA: ", username)
	}
	user := &User{
		Id:       id,
		Username: username,
		Email:    email,
		Dob:      dob,
		Pass:     pass,
		Company:  company,
		Gender:   gender,
	}
	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}
	return user, err
}

func (u *User) Save() error {
	return nil
}

func UpdateUser(u *User) error {
	return nil
}
