package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thanhpk/randstr"
)

type Otp struct {
	Username    string `json:"username" binding:"required"`
	Otp         string `json:"otp" binding:"required"`
	LastUpdated time.Time
	ExpiredTime time.Time
}

func GenerateOTP(username string) (*Otp, error) {
	newOtp := strings.ToUpper(randstr.Hex(4))
	query := session.
		Query(`INSERT INTO user_otp (username, expired_time, last_updated, otp) VALUES (?, ?, ?, ?) IF NOT EXISTS`,
			username,
			time.Now().Add(time.Minute*5),
			time.Now(),
			newOtp,
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

func GetOTPByUsername(uname string) (string, error) {
	var username string
	var expiredTime time.Time
	var lastUpdated time.Time
	var otp string

	err := session.
		Query(`SELECT * FROM user_otp WHERE username = ?`, uname).
		Scan(&username, &expiredTime, &lastUpdated, &otp)		
	if err != nil {
		return "", err
	}

	return otp, nil
}

func GetUserOTP(uname string) (*Otp, error) {
	var username string
	var expiredTime time.Time
	var lastUpdated time.Time
	var otp string

	err := session.
		Query(`SELECT * FROM user_otp WHERE username = ?`, uname).
		Scan(&username, &expiredTime, &lastUpdated, &otp)		
	if err != nil {
		return nil, err
	}

	return &Otp{
		Username: username,
		ExpiredTime: expiredTime,
		LastUpdated: lastUpdated,
		Otp: otp,
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

	user, err := FindUserByUsername(uname)
	if err != nil {
		return err
	}

	query := session.
		Query(`DELETE FROM user_otp WHERE username = ?`, uname)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE user_data_by_id SET is_active = ? WHERE id = ?`, true, user.Id)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE users_by_id SET is_active = ? WHERE id = ?`, true, user.Id)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE users_by_username SET is_active = ? WHERE username = ?`, true, user.Username)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE users_by_email SET is_active = ? WHERE email = ?`, true, user.Email)
	if err = query.Exec(); err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func UpdateOTP(username string) (*Otp, error) {
	newOtp := strings.ToUpper(randstr.Hex(4))
	query := session.
		Query(`UPDATE user_otp SET otp = ?, last_updated = ?, expired_time = ? WHERE username = ?`,
			newOtp,
			time.Now(),
			time.Now().Add(time.Minute*5),
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
