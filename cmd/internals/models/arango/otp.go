package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	"github.com/thanhpk/randstr"
	"strings"
	"time"
)

type Otp struct {
	Username    string    `json:"username" binding:"required"`
	Id          string    `json:"id"`
	Otp         string    `json:"otp" binding:"required"`
	Email       string    `json:"email"`
	IsValidated bool      `json:"isValidated"`
	LastUpdated time.Time `json:"lastUpdated"`
	ExpiredTime time.Time `json:"expiredTime"`
	//DB Info
	CreatedAt time.Time `json:"created_at"`
}

func GenerateOTP(username string, email string) (*Otp, error) {
	newOtp := strings.ToUpper(randstr.Hex(4))
	//createdTime := time.Now()
	//doc := Otp{
	//	Username:    username,
	//	Otp:         newOtp,
	//	Email:       email,
	//	IsValidated: false,
	//	LastUpdated: createdTime,
	//	ExpiredTime: createdTime.Add(time.Minute * 5),
	//	CreatedAt:   createdTime,
	//}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	//otpByUname, _ := FindOTPByUsername(username)
	//if otpByUname != nil {
	//	return nil, &ModelError{
	//		msg:     "duplicated username",
	//		errType: Duplicated,
	//	}
	//}

	query := "UPSERT { username: @username } " +
		"INSERT { username: @username, otp: @newOtp, email: @email, " +
		"lastUpdated: @lastUpdated, expiredTime: @expiredTime, isValidated: @isValidated } " +
		"UPDATE { otp: @newOtp, expiredTime: @expiredTime, lastUpdated: @lastUpdated } IN otps " +
		"RETURN NEW"
	bindVars := map[string]interface{}{
		"newOtp":      newOtp,
		"email":       email,
		"username":    username,
		"expiredTime": time.Now().Add(time.Minute * 5),
		"lastUpdated": time.Now(),
		"isValidated": false,
	}
	otp := Otp{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &otp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		otp.Id = meta.Key
	}

	if otp.Id == "" {
		return nil, &models.ModelError{
			Msg:     "otp not found",
			ErrType: models.DocumentNotFound,
		}
	}
	return &otp, nil
}

func FindOTPById(id string) (*Otp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	otp := Otp{}
	meta, err := otpCol.ReadDocument(ctx, id, &otp)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &models.ModelError{
				Msg:     "otp not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	otp.Id = meta.Key

	return &otp, nil
}

func FindOTPByUsername(uname string) (*Otp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR o IN otps FILTER o.username == @uname LIMIT 1 RETURN o"
	bindVars := map[string]interface{}{
		"uname": uname,
	}

	otp := Otp{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &otp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		otp.Id = meta.Key
	}

	if otp.Id == "" {
		return nil, &models.ModelError{
			Msg:     "otp not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &otp, nil
}
