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
	Otp         string    `json:"otp" binding:"required"`
	Email       string    `json:"email"`
	LastUpdated time.Time `json:"lastUpdated"`
	ExpiredTime time.Time `json:"expiredTime"`
	//DB Info
	CreatedAt time.Time `json:"created_at"`
}

func GenerateOTP(email string) (*Otp, error) {
	newOtp := strings.ToUpper(randstr.Hex(4))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "UPSERT { email: @email } " +
		"INSERT { otp: @newOtp, email: @email, " +
		"lastUpdated: @lastUpdated, expiredTime: @expiredTime } " +
		"UPDATE { otp: @newOtp, expiredTime: @expiredTime, lastUpdated: @lastUpdated } IN otps " +
		"RETURN NEW"
	bindVars := map[string]interface{}{
		"newOtp":      newOtp,
		"email":       email,
		"expiredTime": time.Now().Add(time.Minute * 30),
		"lastUpdated": time.Now(),
	}
	otp := Otp{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &otp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return &otp, nil
}

func OTPConfirm(email string, otp string) error {
	userOtp, err := FindOTPByEmail(email)
	if err != nil {
		return &models.ModelError{
			Msg:     "otp not found",
			ErrType: models.OtpInvalid,
		}
	}

	if userOtp.ExpiredTime.Before(time.Now()) {
		return &models.ModelError{
			Msg:     "otp expired",
			ErrType: models.OtpInvalid,
		}
	}

	if otp != userOtp.Otp {
		return &models.ModelError{
			Msg:     "otp not match",
			ErrType: models.OtpInvalid,
		}
	}

	err = UpdateActive(email, true)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	err = RemoveOTP(userOtp.Email)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return err
}

//func FindOTPByUsername(uname string) (*Otp, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
//	defer cancel()
//
//	query := "FOR o IN otps FILTER o.username == @uname LIMIT 1 RETURN o"
//	bindVars := map[string]interface{}{
//		"uname": uname,
//	}
//
//	otp := Otp{}
//	cursor, err := arangoDb.Query(ctx, query, bindVars)
//	if err != nil {
//		return nil, err
//	}
//	defer cursor.Close()
//
//	for {
//		_, err := cursor.ReadDocument(ctx, &otp)
//		if driver.IsNoMoreDocuments(err) {
//			break
//		} else if err != nil {
//			return nil, err
//		}
//	}
//
//	return &otp, nil
//}

func FindOTPByEmail(email string) (*Otp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR o IN otps FILTER o.email == @email LIMIT 1 RETURN o"
	bindVars := map[string]interface{}{
		"email": email,
	}

	otp := Otp{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &otp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return &otp, nil
}

func RemoveOTP(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR o IN otps FILTER o.email == @email REMOVE o in otps LET removed = OLD RETURN removed"
	bindVars := map[string]interface{}{
		"email": email,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	otp := Otp{}
	for {
		_, err := cursor.ReadDocument(ctx, &otp)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if otp.Otp == "" {
		return &models.ModelError{
			Msg:     "otp not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return nil
}
