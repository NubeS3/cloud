package models

import {
	"error"
	"time"

	"github.com/gocql/gocql"
}

type Otp struct {
	Username   		string
	Otp 					string
	IsConfirmed 	bool
	LastUpdated 	time.Time
	ExpiredTime 	time.Time
}

func SaveOTP(username string, otp string) (string, error) {
	user, err := models.FindUserByUsername(username)
	if err != nil {
		return err
	}

	return username, otp
}