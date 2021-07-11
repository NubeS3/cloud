package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/arangodb/go-driver"
	scrypt "github.com/elithrar/simple-scrypt"
	"time"
)

type User struct {
	Id       string    `json:"id"`
	Fullname string    `json:"fullname"`
	Email    string    `json:"email"`
	Pass     string    `json:"password"`
	Dob      time.Time `json:"dob"`
	IsActive bool      `json:"is_active"`
	IsBanned bool      `json:"is_banned"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type user struct {
	Fullname string    `json:"fullname"`
	Email    string    `json:"email"`
	Pass     string    `json:"password"`
	Dob      time.Time `json:"dob"`
	IsActive bool      `json:"is_active"`
	IsBanned bool      `json:"is_banned"`
	// DB Info
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func SaveUser(
	password string,
	email string,
) (*User, error) {
	u, _ := FindUserByEmail(email)
	if u != nil {
		return nil, &models.ModelError{
			Msg:     "duplicated email",
			ErrType: models.Duplicated,
		}
	}

	createdTime := time.Now()
	passwordHashed, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}

	doc := user{
		Pass:      string(passwordHashed),
		Email:     email,
		IsActive:  false,
		IsBanned:  false,
		CreatedAt: createdTime,
		UpdatedAt: createdTime,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	meta, err := userCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	//LOG CREATE USER
	//_ = nats.SendUserEvent(meta.Key, doc.Username, doc.Email, "Add")

	return &User{
		Id:        meta.Key,
		Pass:      doc.Pass,
		Email:     doc.Email,
		Dob:       doc.Dob,
		IsActive:  doc.IsActive,
		IsBanned:  doc.IsBanned,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

func FindUserById(uid string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	user := User{}
	meta, err := userCol.ReadDocument(ctx, uid, &user)
	if err != nil {
		if driver.IsNotFound(err) {
			return nil, &models.ModelError{
				Msg:     "user not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	user.Id = meta.Key

	return &user, nil
}

func FindUserByUsername(uname string) (*User, error) {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	//defer cancel()
	//
	//query := "FOR u IN users FILTER u.username == @uname LIMIT 1 RETURN u"
	//bindVars := map[string]interface{}{
	//	"uname": uname,
	//}
	//
	//user := User{}
	//cursor, err := arangoDb.Query(ctx, query, bindVars)
	//if err != nil {
	//	return nil, err
	//}
	//defer cursor.Close()
	//
	//for {
	//	meta, err := cursor.ReadDocument(ctx, &user)
	//	if driver.IsNoMoreDocuments(err) {
	//		break
	//	} else if err != nil {
	//		return nil, err
	//	}
	//	user.Id = meta.Key
	//}
	//
	//if user.Id == "" {
	//	return nil, &models.ModelError{
	//		Msg:     "user not found",
	//		ErrType: models.DocumentNotFound,
	//	}
	//}
	//
	//return &user, nil
	return nil, nil
}

func FindUserByEmail(mail string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users FILTER u.email == @email LIMIT 1 RETURN u"
	bindVars := map[string]interface{}{
		"email": mail,
	}

	user := User{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	for {
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		user.Id = meta.Key
	}

	if user.Id == "" {
		return nil, &models.ModelError{
			Msg:     "user not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &user, nil
}

//func UpdateUserData(
//	uid string,
//	fullname string,
//	dob time.Time,
//	company string,
//	gender bool) (*User, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
//	defer cancel()
//
//	updatedTime := time.Now()
//
//	query := "FOR u IN users FILTER u._key == @uid UPDATE u " +
//		"WITH { firstname: @firstname, " +
//		"lastname: @lastname, " +
//		"dob: @dob, " +
//		"company: @company, " +
//		"gender: @gender, " +
//		"updated_at: @updatedAt } " +
//		"IN users RETURN NEW"
//	bindVars := map[string]interface{}{
//		"uid":       uid,
//		"firstname": firstname,
//		"lastname":  lastname,
//		"dob":       dob,
//		"company":   company,
//		"gender":    gender,
//		"updatedAt": updatedTime,
//	}
//
//	cursor, err := arangoDb.Query(ctx, query, bindVars)
//	if err != nil {
//		return nil, &models.ModelError{
//			Msg:     err.Error(),
//			ErrType: models.DbError,
//		}
//	}
//	defer cursor.Close()
//
//	user := User{}
//	for {
//		meta, err := cursor.ReadDocument(ctx, &user)
//		if driver.IsNoMoreDocuments(err) {
//			break
//		} else if err != nil {
//			return nil, &models.ModelError{
//				Msg:     err.Error(),
//				ErrType: models.DbError,
//			}
//		}
//		user.Id = meta.Key
//	}
//
//	if user.Id == "" {
//		return nil, &models.ModelError{
//			Msg:     "folder not found",
//			ErrType: models.DocumentNotFound,
//		}
//	}
//
//	//LOG UPDATE USER
//	//_ = nats.SendUserEvent(user.Id, user.Username, user.Email, "Update")
//
//	return &user, err
//}

func UpdateActive(email string, isActive bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users FILTER u.email == @email " +
		"UPDATE u WITH { is_active: @isActive } IN users RETURN NEW"
	bindVars := map[string]interface{}{
		"email":    email,
		"isActive": isActive,
	}

	user := User{}
	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()

	for {
		_, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return err
		}
	}

	//user, err := FindUserByUsername(uname)
	//if err != nil {
	//	return &models.ModelError{
	//		Msg:     "user not found",
	//		ErrType: models.DocumentNotFound,
	//	}
	//}
	//userUpdate := User{
	//	IsActive: isActive,
	//}
	//
	//meta, err := userCol.UpdateDocument(ctx, user.Id, &userUpdate)
	//
	//if err != nil {
	//	if driver.IsNotFound(err) {
	//		return &models.ModelError{
	//			Msg:     "user not found",
	//			ErrType: models.DocumentNotFound,
	//		}
	//	}
	//
	//	return &models.ModelError{
	//		Msg:     err.Error(),
	//		ErrType: models.DbError,
	//	}
	//}
	//user.Id = meta.Key
	return err
}

func UpdateUserPassword(uid string, password string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
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
			return nil, &models.ModelError{
				Msg:     "user not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	user.Id = meta.Key
	//_ = nats.SendUserEvent(user.Id, user.Username, user.Email, "Update")
	return &user, err
}

func UpdateBanStatus(uid string, isBan bool) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users FILTER u._key == @uid LIMIT 1 UPDATE u WITH { is_banned: @isBan } IN users " +
		"RETURN NEW"
	bindVars := map[string]interface{}{
		"uid":   uid,
		"isBan": isBan,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	user := User{}
	for {
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}
		user.Id = meta.Key
	}

	if user.Id == "" {
		return nil, &models.ModelError{
			Msg:     "user not found",
			ErrType: models.NotFound,
		}
	}
	//_ = nats.SendUserEvent(user.Id, user.Username, user.Email, "Update")
	return &user, nil
}

func RemoveUser(uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	_, err := userCol.RemoveDocument(ctx, uid)
	if err != nil {
		if driver.IsNotFound(err) {
			return &models.ModelError{
				Msg:     "user not found",
				ErrType: models.DocumentNotFound,
			}
		}

		return &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	_ = nats.SendUserEvent(uid, "", "", "Delete")

	return nil
}

func GetAllUser(offset int, limit int) ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users LIMIT @offset, @limit RETURN u"
	bindVars := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	users := []User{}
	for {
		user := user{}
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		users = append(users, User{
			Id:        meta.Key,
			Pass:      user.Pass,
			Email:     user.Email,
			Dob:       user.Dob,
			IsActive:  user.IsActive,
			IsBanned:  user.IsBanned,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return users, nil
}

func GetAllNonBannedUser(offset int, limit int) ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users FILTER u.is_banned == false LIMIT @offset, @limit RETURN u"
	bindVars := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	users := []User{}
	for {
		user := user{}
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		users = append(users, User{
			Id:        meta.Key,
			Pass:      user.Pass,
			Email:     user.Email,
			Dob:       user.Dob,
			IsActive:  user.IsActive,
			IsBanned:  user.IsBanned,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return users, nil
}

func GetAllBannedUser(offset int, limit int) ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users FILTER u.is_banned == true LIMIT @offset, @limit RETURN u"
	bindVars := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}

	cursor, err := arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}
	defer cursor.Close()

	users := []User{}
	for {
		user := user{}
		meta, err := cursor.ReadDocument(ctx, &user)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, &models.ModelError{
				Msg:     err.Error(),
				ErrType: models.DbError,
			}
		}

		users = append(users, User{
			Id:        meta.Key,
			Pass:      user.Pass,
			Email:     user.Email,
			Dob:       user.Dob,
			IsActive:  user.IsActive,
			IsBanned:  user.IsBanned,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return users, nil
}
