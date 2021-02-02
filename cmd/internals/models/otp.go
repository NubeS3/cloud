package models

import (
	"errors"
	"github.com/thanhpk/randstr"
	"strings"
	"time"
)

type Otp struct {
	Username    string `json:"username" binding:"required"`
	Otp         string `json:"otp" binding:"required"`
	LastUpdated time.Time
	ExpiredTime time.Time
}

func GenerateOTP(username string) (string, error) {
	newOtp := strings.ToUpper(randstr.Hex(6))
	query := session.
		Query(`INSERT INTO user_otp VALUES (?, ?, ?, ?)`,
			username,
			newOtp,
			time.Now(),
			time.Now().Add(time.Minute*5),
		)
	if err := query.Exec(); err != nil {
		return "", err
	}
	return newOtp, nil
}

func GetOTPByUsername(uname string) (string, error) {
	var otps []Otp = []Otp{}
	var otp *Otp

	iter := session.
		Query(`SELECT * FROM user_otp WHERE username = ?`, uname).
		Iter()

	for iter.Scan(&uname) {
		otp = &Otp{
			Username: uname,
		}
		otps = append(otps, *otp)
	}

	if err := iter.Close(); err != nil {
		return "", errors.New("otp not found")
	}

	return *&otps[0].Otp, nil
}

func GetUserOTP(uname string) (*Otp, error) {
	var otps []Otp = []Otp{}
	var otp *Otp

	iter := session.
		Query(`SELECT * FROM user_otp WHERE username = ?`, uname).
		Iter()

	for iter.Scan(&uname) {
		otp = &Otp{
			Username: uname,
		}
		otps = append(otps, *otp)
	}

	if err := iter.Close(); err != nil {
		return nil, errors.New("otp not found")
	}

	return &otps[0], nil
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
		Query(`DELETE FROM user_otp WHERE username = ?`, uname)

	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE user_data_by_id SET is_active = ?`, true)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE user_by_id SET is_active = ?`, true)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE user_by_username SET is_active = ?`, true)
	if err = query.Exec(); err != nil {
		return err
	}

	query = session.
		Query(`UPDATE user_by_email SET is_active = ?`, true)
	if err = query.Exec(); err != nil {
		return err
	}

	return nil
}

func UpdateOTP(username string) (string, error) {
	newOtp := strings.ToUpper(randstr.Hex(6))
	query := session.
		Query(`UPDATE user_otp SET otp = ?, is_confirmed = ?, last_updated = ?, expired_time = ?`,
			newOtp,
			time.Now(),
			time.Now().Add(time.Minute*5),
		)
	if err := query.Exec(); err != nil {
		return "", err
	}
	return newOtp, nil
}
