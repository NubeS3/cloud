package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/arangodb/go-driver"
	"github.com/thanhpk/randstr"
	"time"
)

type RefreshToken struct {
	Uid         string    `json:"uid"`
	RfToken     string    `json:"rfToken"`
	ExpiredTime time.Time `json:"expiredTime"`
	//DB Info
	CreatedAt time.Time `json:"created_at"`
}

func GenerateRfToken(uid string) error {
	newRfToken := randstr.Hex(8)

	doc := RefreshToken{
		Uid:         uid,
		RfToken:     newRfToken,
		ExpiredTime: time.Now().Add(time.Hour * 4320),
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	rfToken, _ := FindRfTokenByUid(uid)
	if rfToken != nil {
		return &models.ModelError{
			Msg:     "duplicated rfToken",
			ErrType: models.Duplicated,
		}
	}

	_, err := rfTokenCol.CreateDocument(ctx, doc)
	if err != nil {
		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	return nil
}

func FindRfTokenByUid(uid string) (*RefreshToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR r IN rfTokens FILTER r.uid == @uid LIMIT 1 RETURN r"
	bindVars := map[string]interface{}{
		"uid": uid,
	}

	rfToken := RefreshToken{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &rfToken)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
	}

	if rfToken.RfToken == "" {
		return nil, &models.ModelError{
			Msg:     "rfToken not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &rfToken, nil
}

func UpdateToken(uid string) (*string, *string, error) {
	newAccessToken, err := ultis.CreateToken(uid)
	if err != nil {
		return nil, nil, &models.ModelError{
			Msg:     "token create failed.",
			ErrType: models.TokenInvalid,
		}
	}
	rfToken := randstr.Hex(8)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR r IN rfTokens FILTER r.uid == @uid " +
		"UPDATE r WITH { rfToken: @rfToken, expiredTime: @expiredTime } IN rfTokens RETURN NEW"
	bindVars := map[string]interface{}{
		"uid":         uid,
		"rfToken":     rfToken,
		"expiredTime": time.Now().Add(time.Hour * 4320),
	}

	rfTokenUpdate := RefreshToken{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &rfTokenUpdate)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
	}

	return &newAccessToken, &rfToken, err
}
