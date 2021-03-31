package arango

import (
	"context"
	"github.com/NubeS3/cloud/cmd/internals/models"
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

type user struct {
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
	createdTime := time.Now()
	passwordHashed, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}

	doc := user{
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	user, _ := FindUserByUsername(username)
	if user != nil {
		return nil, &models.ModelError{
			Msg:     "duplicated username",
			ErrType: models.Duplicated,
		}
	}

	meta, err := userCol.CreateDocument(ctx, doc)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.DbError,
		}
	}

	return &User{
		Id:        meta.Key,
		Firstname: doc.Firstname,
		Lastname:  doc.Lastname,
		Username:  doc.Username,
		Pass:      doc.Pass,
		Email:     doc.Email,
		Dob:       doc.Dob,
		Company:   doc.Company,
		Gender:    doc.Gender,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
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
		return nil, &models.ModelError{
			Msg:     "user not found",
			ErrType: models.DocumentNotFound,
		}
	}

	return &user, nil
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

func UpdateUserData(
	uid string,
	firstname string,
	lastname string,
	dob time.Time,
	company string,
	gender bool) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	isTimeZero := dob.IsZero()
	updatedTime := time.Now()

	query := "FOR u IN users FILTER u._key == @uid UPDATE u " +
		"WITH { firstname: CHAR_LENGTH(@firstname) == 0 ? u.firstname : @firstname, " +
		"lastname: CHAR_LENGTH(@firstname) == 0 ? u.lastname : @lastname, " +
		"dob: @isTimeZero == true ? u.dob : @dob, " +
		"company: CHAR_LENGTH(@firstname) == 0 ? u.company : @company, " +
		"gender: @gender == u.gender ? u.gender : @gender, updated_at: @updatedAt } " +
		"IN users RETURN NEW"
	bindVars := map[string]interface{}{
		"uid":        uid,
		"firstname":  firstname,
		"lastname":   lastname,
		"dob":        dob,
		"isTimeZero": isTimeZero,
		"company":    company,
		"gender":     gender,
		"updatedAt":  updatedTime,
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
			Msg:     "folder not found",
			ErrType: models.DocumentNotFound,
		}
	}
	return &user, err
}

func UpdateActive(uname string, isActive bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	query := "FOR u IN users FILTER u.username == @uname " +
		"UPDATE u WITH { is_active: @isActive } IN users RETURN NEW"
	bindVars := map[string]interface{}{
		"uname":    uname,
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
	return &user, err
}

func UpdateBanStatus(uid string, isBan bool) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	user := User{
		IsBanned: isBan,
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
	return &user, err
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

	return nil
}
