package models

import (
	"errors"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/thanhpk/randstr"
)

type Otp struct {
	Username    string `json:"username" binding:"required"`
	Id 					gocql.UUID
	Otp         string `json:"otp" binding:"required"`
	Email 			string
	IsValidated bool
	LastUpdated time.Time
	ExpiredTime time.Time
}

func GenerateOTP(username string, id gocql.UUID, email string) (*Otp, error) {
	newOtp := strings.ToUpper(randstr.Hex(4))
	now := time.Now()
	otp := &Otp{
		Username: username,
		Id: id,
		Otp: newOtp,
		Email: email,
		LastUpdated: now,
		ExpiredTime: now.Add(time.Minute * 5),
		IsValidated: false,
	}
	query := session.
		Query(`INSERT INTO user_otp (username, email, expired_time, id, is_validated, last_updated, otp) VALUES (?, ?, ?, ?) IF NOT EXISTS`,
			otp.Username,
			otp.Email,
			otp.ExpiredTime,
			otp.Id,
			otp.IsValidated,
			otp.LastUpdated,
			otp.Otp,
		)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	return otp, nil
}

func GetOTPByUsername(uname string) (string, error) {
	var username string
	var email string
	var expiredTime time.Time
	var id string
	var isValidated string
	var lastUpdated time.Time
	var otp string

	err := session.
		Query(`SELECT * FROM user_otp WHERE username = ? LIMIT 1`, uname).
		Scan(&username, &email, &expiredTime, &id, &isValidated, &lastUpdated, &otp)

	if err != nil {
		return "", err
	}

	return otp, nil
}

func GetUserOTP(uname string) (*Otp, error) {
	var username string
	var email string
	var expiredTime time.Time
	var id gocql.UUID
	var isValidated bool
	var lastUpdated time.Time
	var otp string

	err := session.
		Query(`SELECT * FROM user_otp WHERE username = ? LIMIT 1`, uname).
		Scan(&username, &email, &expiredTime, &id, &isValidated, &lastUpdated, &otp)

	if err != nil {
		return nil, err
	}

	return &Otp{
		Username:    username,
		ExpiredTime: expiredTime,
		Id: id,
		Email: email,
		LastUpdated: lastUpdated,
		Otp: otp,
		IsValidated: isValidated,
	}, nil
}

func OTPConfirm(uname string, otp string) error {
	userOtp, err := GetUserOTP(uname)
	if err != nil {
		return errors.New("otp not found")
	}

	if userOtp.ExpiredTime.Before(time.Now()) {
		return errors.New("otp is expired")
	}

	if otp != userOtp.Otp {
		return errors.New("otp not match")
	}

	query := session.
		Query(`UPDATE user_otp SET is_validated = ? WHERE username = ?`, true, uname)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE user_data_by_id SET is_active = ? WHERE id = ?`, true, userOtp.Id)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE users_by_id SET is_active = ? WHERE id = ?`, true, userOtp.Id)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE users_by_username SET is_active = ? WHERE username = ?`, true, uname)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE users_by_email SET is_active = ? WHERE email = ?`, true, userOtp.Email)
	if err = query.Exec(); err != nil {
		return err
	}

	return nil
}

func ReGenerateOTP(username string) (*Otp, error) {
	newOtp := strings.ToUpper(randstr.Hex(4))
	now := time.Now()
	query := session.
		Query(`UPDATE user_otp SET otp = ?, last_updated = ?, expired_time = ? WHERE username = ?`,
			newOtp,
			now,
			now.Add(time.Minute*5),
			username,
		)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	otp, err := GetUserOTP(username)
	if err != nil {
		return nil, err
	}

	return otp, nil
}
