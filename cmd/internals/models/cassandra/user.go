package cassandra

import (
	"time"

	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gocql/gocql"
)

type User struct {
	Id        gocql.UUID `json:"id"`
	Firstname string     `json:"firstname" binding:"required"`
	Lastname  string     `json:"lastname" binding:"required"`
	Username  string     `json:"username" binding:"required"`
	Pass      string     `json:"password" binding:"required"`
	Email     string     `json:"email" binding:"required"`
	Dob       time.Time  `json:"dob" binding:"required"`
	Company   string     `json:"company" binding:"required"`
	Gender    bool       `json:"gender" binding:"required"`
	IsActive  bool
	IsBanned  bool
	// DB Info
	CreatedAt time.Time
	UpdatedAt time.Time
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
	id, err := gocql.RandomUUID()
	if err != nil {
		return nil, err
	}

	passwordHashed, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	query := session.
		Query(`INSERT INTO user_data_by_id (
			id, 
			company, 
			created_at, 
			dob, 
			email, 
			firstname, 
			gender, 
			is_active, 
			is_banned, 
			lastname, 
			updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id,
			company,
			now,
			dob,
			email,
			firstname,
			gender,
			false,
			false,
			lastname,
			now,
		)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`INSERT INTO users_by_email (
			email, 
			id, 
			is_active, 
			is_banned, 
			password, 
			username) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			email,
			id,
			false,
			false,
			passwordHashed,
			username,
		)
	if err := query.Exec(); err != nil {
		session.Query(`DELETE FROM user_data_by_id WHERE id = ?`, id).Exec()
		return nil, err
	}

	query = session.
		Query(`INSERT INTO users_by_username (
			username, 
			id, 
			is_active, 
			is_banned, 
			password) 
			VALUES (?, ?, ?, ?, ?)`,
			username,
			id,
			false,
			false,
			passwordHashed,
		)
	if err := query.Exec(); err != nil {
		session.Query(`DELETE FROM user_data_by_id WHERE id = ?`, id).Exec()
		session.Query(`DELETE FROM users_by_email WHERE id = ?`, id).Exec()
		return nil, err
	}

	query = session.
		Query(`INSERT INTO users_by_id (
			id,
			is_active, 
			is_banned, 
			password,
			username) 
			VALUES (?, ?, ?, ?, ?)`,
			id,
			false,
			false,
			passwordHashed,
			username,
		)
	if err := query.Exec(); err != nil {
		session.Query(`DELETE FROM user_data_by_id WHERE id = ?`, id).Exec()
		session.Query(`DELETE FROM users_by_email WHERE id = ?`, id).Exec()
		session.Query(`DELETE FROM users_by_username WHERE id = ?`, id).Exec()
		return nil, err
	}

	user := &User{
		Id:        id,
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Pass:      string(passwordHashed),
		Email:     email,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
	}

	return user, err
}

func FindUserById(uid gocql.UUID) (*User, error) {
	var id gocql.UUID
	var firstname string
	var lastname string
	var username string
	var pass string
	var email string
	var dob time.Time
	var company string
	var gender bool
	var isActive bool
	var isBanned bool
	var createdAt time.Time
	var updatedAt time.Time

	err := session.
		Query(`SELECT * FROM users_by_id WHERE id = ? LIMIT 1`, uid).
		Scan(&id, &isActive, &isBanned, &pass, &username)
	if err != nil {
		return nil, err
	}

	err = session.
		Query(`SELECT * FROM user_data_by_id WHERE id = ? LIMIT 1`, uid).
		Scan(&id, &company, &createdAt, &dob, &email, &firstname, &gender, &isActive, &isBanned, &lastname, &updatedAt)
	if err != nil {
		return nil, err
	}

	return &User{
		Id:        uid,
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Pass:      pass,
		Email:     email,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
		// IsActive:  isActive,
		// IsBanned:  isBanned,
		// CreatedAt: createdAt,
		// UpdatedAt: updatedAt,
	}, nil

}

func FindUserByUsername(uname string) (*User, error) {
	var id gocql.UUID
	var firstname string
	var lastname string
	var username string
	var pass string
	var email string
	var dob time.Time
	var company string
	var gender bool
	var isActive bool
	var isBanned bool
	var createdAt time.Time
	var updatedAt time.Time

	err := session.
		Query(`SELECT * FROM users_by_username WHERE username = ? LIMIT 1`, uname).
		Scan(&username, &id, &isActive, &isBanned, &pass)
	if err != nil {
		return nil, err
	}

	err = session.
		Query(`SELECT * FROM user_data_by_id WHERE id = ? LIMIT 1`, id).
		Scan(&id, &company, &createdAt, &dob, &email, &firstname, &gender, &isActive, &isBanned, &lastname, &updatedAt)
	if err != nil {
		return nil, err
	}

	return &User{
		Id:        id,
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Pass:      pass,
		Email:     email,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
		// IsActive:  isActive,
		// IsBanned:  isBanned,
		// CreatedAt: createdAt,
		// UpdatedAt: updatedAt,
	}, nil

}

func FindUserByEmail(mail string) (*User, error) {
	var id gocql.UUID
	var firstname string
	var lastname string
	var username string
	var pass string
	var email string
	var dob time.Time
	var company string
	var gender bool
	var isActive bool
	var isBanned bool
	var createdAt time.Time
	var updatedAt time.Time

	err := session.
		Query(`SELECT * FROM users_by_email WHERE email = ? LIMIT 1`, mail).
		Scan(&email, &id, &isActive, &isBanned, &pass, &username)
	if err != nil {
		return nil, err
	}

	err = session.
		Query(`SELECT * FROM user_data_by_id WHERE id = ? LIMIT 1`, id).
		Scan(&id, &company, &createdAt, &dob, &email, &firstname, &gender, &isActive, &isBanned, &lastname, &updatedAt)
	if err != nil {
		return nil, err
	}

	return &User{
		Id:        id,
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Pass:      pass,
		Email:     email,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
		// IsActive:  isActive,
		// IsBanned:  isBanned,
		// CreatedAt: createdAt,
		// UpdatedAt: updatedAt,
	}, nil

}

func UpdateUserData(
	uid gocql.UUID,
	firstname string,
	lastname string,
	dob time.Time,
	company string,
	gender bool) (*User, error) {
	_, err := FindUserById(uid)
	if err != nil {
		return nil, err
	}

	query := session.
		Query(`UPDATE user_data_by_id SET firstname = ?, lastname = ?, dob = ?, company = ?, gender = ? WHERE id = ?`,
			firstname,
			lastname,
			dob,
			company,
			gender,
			uid,
		)

	if err := query.Exec(); err != nil {
		return nil, err
	}

	user, err := FindUserById(uid)
	if err != nil {
		return nil, err
	}

	return user, err
}

func UpdateUserPassword(uid gocql.UUID, password string) (*User, error) {
	_, err := FindUserById(uid)
	if err != nil {
		return nil, err
	}

	query := session.
		Query(`UPDATE users_by_id SET password = ?`, password)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`UPDATE users_by_username SET password = ?`, password)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	query = session.
		Query(`UPDATE users_by_email SET password = ?`, password)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	user, err := FindUserById(uid)

	return user, err
}

func RemoveUserById(uid gocql.UUID) []error {
	var errors []error
	_, err := FindUserById(uid)

	if err != nil {
		errors = append(errors, err)
	}

	query := session.
		Query(`DELETE FROM users_by_id WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		errors = append(errors, err)
	}

	query = session.
		Query(`DELETE FROM users_by_username WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		errors = append(errors, err)
	}

	query = session.
		Query(`DELETE FROM user_data_by_id WHERE id = ?`, uid)
	if err := query.Exec(); err != nil {
		errors = append(errors, err)
	}

	if len(errors) < 1 {
		return nil
	}

	return errors
}
