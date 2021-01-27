package ultis

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gocql/gocql"
	"time"
)

type UserClaims struct {
	Id gocql.UUID
	jwt.StandardClaims
}

func CreateToken(oid gocql.UUID) (string, error) {
	userClaims := UserClaims{
		Id: oid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 3600).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, userClaims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func ParseToken(authToken string, claims *UserClaims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(authToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("wrong method")
		}
		return []byte(secret), nil
	})
}
