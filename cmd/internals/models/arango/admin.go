package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/arangodb/go-driver"
	scrypt "github.com/elithrar/simple-scrypt"
	"time"
)

const (
	RootAdmin = iota
	ModAdmin
)

type AdminType int

type Admin struct {
	Id        string    `json:"id"`
	Username  string    `json:"username" binding:"required"`
	Pass      string    `json:"password" binding:"required"`
	IsDisable bool      `json:"is_disabled"`
	AType     AdminType `json:"type"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CreateAdmin(
	username string,
	password string,
	aType AdminType,
) (*Admin, error) {
	createdTime := time.Now()
	passwordHashed, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}

	doc := Admin{
		Username:  username,
		Pass:      string(passwordHashed),
		AType:     aType,
		IsDisable: false,
		CreatedAt: createdTime,
		UpdatedAt: createdTime,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	admin, _ := FindAdminByUsername(username)
	if admin != nil {
		return nil, &models.ModelError{
			Msg:     "duplicated username",
			ErrType: models.Duplicated,
		}
	}

	meta, err := adminCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	//LOG
	doc.Id = meta.Key

	return &doc, nil
}

func FindAdminByUsername(username string) (*Admin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR a IN admin FILTER a.username == @uname LIMIT 1 RETURN a"
	bindVars := map[string]interface{}{
		"uname": username,
	}

	admin := Admin{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &admin)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		admin.Id = meta.Key
	}

	if admin.Id == "" {
		return nil, &models.ModelError{
			Msg:     "user not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &admin, nil
}

func FindAdminById(id string) (*Admin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR a IN admin FILTER a._key == @id LIMIT 1 RETURN a"
	bindVars := map[string]interface{}{
		"id": id,
	}

	admin := Admin{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &admin)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		admin.Id = meta.Key
	}

	if admin.Id == "" {
		return nil, &models.ModelError{
			Msg:     "user not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &admin, nil
}

func ToggleAdmin(username string, disable bool) (*Admin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR a IN admin FILTER a.username == @uname && a.type != 0 LIMIT 1 UPDATE { _key: a._key, is_disabled: @disable } IN admin RETURN NEW"
	bindVars := map[string]interface{}{
		"uname":   username,
		"disable": disable,
	}

	admin := Admin{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &admin)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		admin.Id = meta.Key
	}

	if admin.Id == "" {
		return nil, &models.ModelError{
			Msg:     "user not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &admin, nil
}
