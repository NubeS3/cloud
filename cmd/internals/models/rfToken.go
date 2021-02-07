package models

import (
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gocql/gocql"
	"github.com/thanhpk/randstr"
	"time"
)

type RefreshToken struct {
	Uid         gocql.UUID
	RfToken     string
	ExpriedTime time.Time
}

func FindRfTokenByUid(uid gocql.UUID) (*RefreshToken, error) {
	var rfToken string
	var expiredRf time.Time
	err := session.
		Query(`SELECT * FROM refresh_token WHERE uid = ?`, uid).
		Scan(&uid, &expiredRf, &rfToken)
	if err != nil {
		return nil, err
	}
	return &RefreshToken{
		Uid:         uid,
		RfToken:     rfToken,
		ExpriedTime: expiredRf,
	}, nil
}

func UpdateToken(uid gocql.UUID) (*string, *string, error) {
	newAccessToken, err := ultis.CreateToken(uid)
	if err != nil {
		return nil, nil, err
	}

	rfToken := randstr.Hex(8)
	query := session.
		Query(`UPDATE refresh_token SET rf_token = ?, expired_rf = ? WHERE uid = ?`,
			rfToken,
			time.Now().Add(time.Hour*4320),
			uid,
		)

	if err := query.Exec(); err != nil {
		return nil, nil, err
	}

	return &newAccessToken, &rfToken, nil
}

func GenerateRfToken(uid gocql.UUID) error {
	rfToken := randstr.Hex(8)
	query := session.
		Query(`UPDATE refresh_token SET rf_token = ?, expired_rf = ? WHERE uid = ?`,
			rfToken,
			time.Now().Add(time.Hour*4320),
			uid,
		)

	if err := query.Exec(); err != nil {
		return err
	}
	return nil
}
