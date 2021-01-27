package models

import (
	"errors"
	"log"
	"time"
	"github.com/gocql/gocql"
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

type UserById struct {
	Id           gocql.UUID
	Username     string
	Pass         string
}

func SaveUser(firstname string, lastname string, username string, password string, email string, dob time.Time, company string, gender bool) (*User, error) {
	id, err := gocql.RandomUUID()
	if err != nil {
		return nil, err
	}
	query := session.
		Query(`INSERT INTO user_data_by_id VALUES (?, ? ,?, ?, ?, ?, ?) IF NOT EXIST`,
			id,
			company,
			dob,
			email,
			firstname,
			gender,
			lastname,
		)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`INSERT INTO users_by_username VALUES (?, ?, ?) IF NOT EXIST`,
			username,
			id,
			password,
		)
	if err := query.Exec(); err != nil {
		session.Query(`DELETE FROM user_data_by_id WHERE id = ?`, id)
		return nil, err
	}

	query = session.
		Query(`INSERT INTO users_by_id VALUES (?, ?, ?)`,
			id,
			password,
			username,
		)
	if err := query.Exec(); err != nil {
		session.Query(`DELETE FROM user_data_by_id WHERE id = ?`, id)
		session.Query(`DELETE FROM users_by_username WHERE id = ?`, id)
		return nil, err
	}

	user := &User {
		Id: id,
		Firstname: firstname,
		Lastname: lastname,
		Username: username,
		Pass: password,
		Email: email,
		Dob: dob,
		Company: company,
		Gender: gender,
	}

	return user, nil
}

func FindUserById(uid gocql.UUID) (*User, error) {
	var users []User
	users = []User{}

	var user *User
	iter := session.
		Query(`SELECT * FROM users_by_id WHERE id = ?`, uid).
		Iter()

	var id gocql.UUID
	var username string
	var password string

	for iter.Scan(&id, &password, &username) {
		user = &User {
			Id: id,
			Username: username,
			Pass: password,
		}
		users = append(users, *user)
	}

	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(users) < 1 {
		return nil, errors.New("user not found")
	}

	return &users[0], nil
}

func FindUserByUsername(uname string) (*User, error) {
	var users []User
	users = []User{}

	var user *User
	iter := session.
		Query(`SELECT * FROM users_by_username WHERE username = ? `, uname).
		Iter()

	var username string
	var id gocql.UUID
	var password string

	for iter.Scan(&username, &id, &password) {
		user = &User {
			Id: id,
			Username: username,
			Pass: password,
		}
		users = append(users, *user)
	}

	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(users) < 1 {
		return nil, errors.New("user not found")
	}

	return &users[0], nil
}

func FindUserByEmail(mail string) (*User, error) {
	var users []User
	users = []User{}

	var user *User
	iter := session.
		Query(`SELECT * FROM users_by_email WHERE email = ?`, mail).
		Iter()

	var id gocql.UUID
	var username string
	var email string

	for iter.Scan(&email, &id, &username) {
		user = &User {
			Id: id,
			Username: username,
			Email: email,
		}
		users = append(users, *user)
	}

	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(users) < 1 {
		return nil, errors.New("user not found")
	}

	return &users[0], nil
}

func UpdateUser() {
	
}

func RemoveUserById(uid gocql.UUID) error {
	user, err := FindUserById(uid)

	if user == nil && err != nil {
		return err
	}

	query := session.
		Query(`DELETE FROM users_by_id WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`DELETE FROM users_by_username WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`DELETE FROM user_data_by_id WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		return err
	}

	return nil
}