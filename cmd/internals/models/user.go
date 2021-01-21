package models

import "time"

type User struct {
	Id           string
	Firstname    string
	Lastname     string
	Username     string
	Pass         string
	Email        string
	Dob          time.Time
	RefreshToken string
	ExpiredRf    time.Time
	IsBanned     bool
	// DB Info
	CreatedAt time.Time
	UpdatedAt time.Time
}

func FindUserById(id string) (*User, error) {
	return nil, nil
}

func FindUserByUsername(username string) (*User, error) {
	return nil, nil
}

func SaveUser(u *User) error {
	return nil
}

func UpdateUser(u *User) error {
	return nil
}
