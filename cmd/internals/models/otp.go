package models

import (
	"github.com/gocql/gocql"
	"github.com/thanhpk/randstr"
	"strings"
	"time"
)

type Otp struct {
	Uid         gocql.UUID
	Otp         string
	IsConfirmed bool
	LastUpdated time.Time
	ExpiredTime time.Time
}

func GenerateOTP(uid gocql.UUID) error {
	newOtp := strings.ToUpper(randstr.Hex(6))
	query := session.
		Query(`INSERT INTO user_otp VALUES (?, ?, ?, ?, ?)`,
			uid,
			newOtp,
			false,
			time.Now(),
			time.Now().Add(time.Minute*5),
		)
	if err := query.Exec(); err != nil {
		return err
	}
	return nil
}
