package arango

import (
	"context"
	"github.com/arangodb/go-driver"
	scrypt "github.com/elithrar/simple-scrypt"
	"time"
)

type User struct {
	Id        string    `json:"id"`
	Firstname string    `json:"firstname" binding:"required"`
	Lastname  string    `json:"lastname" binding:"required"`
	Username  string    `json:"username" binding:"required"`
	Pass      string    `json:"password" binding:"required"`
	Email     string    `json:"email" binding:"required"`
	Dob       time.Time `json:"dob" binding:"required"`
	Company   string    `json:"company" binding:"required"`
	Gender    bool      `json:"gender" binding:"required"`
	IsActive  bool      `json:"is_active"`
	IsBanned  bool      `json:"is_banned"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func SaveUser(
	firstname string,
	lastname string,
	username string,
	password string,
	email string,
	dob time.Time,
	company string,
	gender bool,
) (*User, error) {
	createdTime := time.Time{}
	passwordHashed, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}

	doc := User{
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Pass:      string(passwordHashed),
		Email:     email,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
		IsActive:  false,
		IsBanned:  false,
		CreatedAt: createdTime,
		UpdatedAt: createdTime,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	user, _ := FindUserByUsername(username)
	if user != nil {
		return nil, &ModelError{
			msg:     "duplicated username",
			errType: Duplicated,
		}
	}

	meta, err := userCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	doc.Id = meta.Key
	return &doc, nil
}

func FindUserById(uid string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	user := User{}
	meta, err := userCol.ReadDocument(ctx, uid, &user)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &ModelError{
				msg:     "user not found",
				errType: DocumentNotFound,
			}
		}

		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	user.Id = meta.Key

	return &user, nil
}

func FindUserByUsername(uname string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR u IN users FILTER u.username == @uname LIMIT 1 RETURN u"
	bindVars := map[string]interface{}{
		"uname": uname,
	}

	user := User{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		user.Id = meta.Key
	}

	if user.Id == "" {
		return nil, &ModelError{
			msg:     "user not found",
			errType: DocumentNotFound,
		}
	}

	return &user, nil
}

func FindUserByEmail(mail string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "FOR u IN users FILTER u.email == @email LIMIT 1 RETURN u"
	bindVars := map[string]interface{}{
		"email": mail,
	}

	user := User{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		user.Id = meta.Key
	}

	if user.Id == "" {
		return nil, &ModelError{
			msg:     "user not found",
			errType: DocumentNotFound,
		}
	}

	return &user, nil
}

func UpdateUserData(
	uid string,
	firstname string,
	lastname string,
	dob time.Time,
	company string,
	gender bool) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	user := User{
		Firstname: firstname,
		Lastname:  lastname,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
	}

	meta, err := userCol.UpdateDocument(ctx, uid, &user)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &ModelError{
				msg:     "user not found",
				errType: DocumentNotFound,
			}
		}

		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	user.Id = meta.Key
	return &user, err
}

func UpdateUserPassword(uid string, password string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	passwordHashed, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}
	user := User{
		Pass: string(passwordHashed),
	}

	meta, err := userCol.UpdateDocument(ctx, uid, &user)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &ModelError{
				msg:     "user not found",
				errType: DocumentNotFound,
			}
		}

		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	user.Id = meta.Key
	return &user, err
}

func UpdateBanStatus(uid string, isBan bool) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	user := User{
		IsBanned: isBan,
	}

	meta, err := userCol.UpdateDocument(ctx, uid, &user)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &ModelError{
				msg:     "user not found",
				errType: DocumentNotFound,
			}
		}

		return nil, &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	user.Id = meta.Key
	return &user, err
}

func RemoveUser(uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := userCol.RemoveDocument(ctx, uid)
	if err != nil {
		if driver.IsNotFound(err) {
			return &ModelError{
				msg:     "user not found",
				errType: DocumentNotFound,
			}
		}

		return &ModelError{
			msg:     err.Error(),
			errType: DbError,
		}
	}

	return nil
}
