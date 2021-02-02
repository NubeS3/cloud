package models

import (
	"errors"
	"log"
	"time"

	"github.com/gocql/gocql"
)

type User struct {
	field.DefaultField 			`json:",inline" bson:",inline"`
	Id           gocql.UUID `json:"id" bson:"id" binding:"required"`
	Firstname    string 		`json:"firstname" bson:"firstname" binding:"required"`
	Lastname     string 		`json:"lastname" bson:"lastname" binding:"required"`
	Username     string 		`json:"username" bson:"username" binding:"required"`
	Pass         string 		`json:"pass" bson:"pass" binding:"required"`
	Email        string 		`json:"email" bson:"email" binding:"required"`
	Dob          time.Time 	`json:"dob" bson:"dob"`
	Company      string 		`json:"company" bson:"company"`
	Gender       bool 			`json:"gender" bson:"gender"`
	RefreshToken string 		`json:"refreshToken" bson:"refreshToken" binding:"required"`
	ExpiredRf    time.Time 	`json:"expiredRf" bson:"expiredRf" binding:"required"`
	IsActive 	   bool 			`json:"isActive" bson:"isActive" binding:"required"`
	IsBanned     bool 			`json:"isBanned" bson:"isBanned" binding:"required"`
	// DB Info
	CreatedAt time.Time 		`json:"createdAt" bson:"createdAt" binding:"required"`
	UpdatedAt time.Time 		`json:"updatedAt" bson:"updatedAt " binding:"required"`
}

func SaveUser(
	firstname string,
	lastname string,
	username string,
	password string,
	email string,
	dob time.Time,
	company string,
	gender bool,
) (*User, error) {
	id, err := gocql.RandomUUID()
	if err != nil {
		return nil, err
	}

	query := session.
		Query(`INSERT INTO user_data_by_id VALUES (?, ? ,?, ?, ?, ?, ?, ?)`,
			id,
			company,
			dob,
			email,
			firstname,
			gender,
			false,
			lastname,
		)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`INSERT INTO users_by_username VALUES (?, ?, ?)`,
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

	user, err := FindUserById(id)
	if err != nil {
		return nil, err
	}

	return user, err
}

func FindUserById(uid gocql.UUID) (*User, error) {
	var users []User = []User{}

	var user *User
	iter := session.
		Query(`SELECT * FROM users_by_id WHERE id = ?`, uid).
		Iter()

	var id gocql.UUID
	var username string
	var password string

	for iter.Scan(&id, &password, &username) {
		user = &User{
			Id:       id,
			Username: username,
			Pass:     password,
		}
		users = append(users, *user)
	}

	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(users) < 1 {
		return nil, errors.New("User not found")
	}

	return &users[0], nil
}

func FindUserByUsername(uname string) (*User, error) {
	var users []User

	var user *User
	iter := session.
		Query(`SELECT * FROM users_by_username WHERE username = ?`, uname).
		Iter()

	var username string
	var id gocql.UUID
	var password string

	for iter.Scan(&username, &id, &password) {
		user = &User{
			Id:       id,
			Username: username,
			Pass:     password,
		}
		users = append(users, *user)
	}

	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(users) < 1 {
		return nil, errors.New("User not found")
	}

	return &users[0], nil
}

func FindUserByEmail(mail string) (*User, error) {
	var users []User

	var user *User
	iter := session.
		Query(`SELECT * FROM users_by_email WHERE email = ?`, mail).
		Iter()

	var id gocql.UUID
	var username string
	var email string

	for iter.Scan(&email, &id, &username) {
		user = &User{
			Id:       id,
			Username: username,
			Email:    email,
		}
		users = append(users, *user)
	}

	var err error
	if err = iter.Close(); err != nil {
		log.Fatal(err)
	}

	if len(users) < 1 {
		return nil, errors.New("User not found")
	}

	return &users[0], nil
}

func UpdateUserData(
	uid gocql.UUID,
	firstname string,
	lastname string,
	dob time.Time,
	company string,
	gender bool) (*User, error) {
	_, err := FindUserById(uid)
	if err != nil {
		return nil, err
	}

	query := session.
		Query(`UPDATE user_data_by_id SET firstname = ?, lastname = ?, dob = ?, company = ?, gender = ? WHERE id = ?`,
			firstname,
			lastname,
			dob,
			company,
			gender,
			uid,
		)

	if err := query.Exec(); err != nil {
		return nil, err
	}

	user, err := FindUserById(uid)

	return user, err
}

func UpdateUserPassword(uid gocql.UUID, password string) (*User, error) {
	_, err := FindUserById(uid)
	if err != nil {
		return nil, err
	}

	query := session.
		Query(`UPDATE users_by_id SET password = ?`, password)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`UPDATE users_by_username SET password = ?`, password)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`UPDATE users_by_email SET password = ?`, password)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	user, err := FindUserById(uid)

	return user, err
}

func RemoveUserById(uid gocql.UUID) []error {
	var errors []error
	_, err := FindUserById(uid)

	if err != nil {
		errors = append(errors, err)
	}

	query := session.
		Query(`DELETE FROM users_by_id WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		errors = append(errors, err)
	}

	query = session.
		Query(`DELETE FROM users_by_username WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		errors = append(errors, err)
	}

	query = session.
		Query(`DELETE FROM user_data_by_id WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		errors = append(errors, err)
	}

	if len(errors) < 1 {
		return nil
	}

	return errors
}
