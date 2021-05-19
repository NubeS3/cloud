package ultis

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type UserClaims struct {
	Id string
	jwt.StandardClaims
}

type AdminClaims struct {
	Id        string
	AdminType int
	jwt.StandardClaims
}

type KeyClaims struct {
	KeyId string
	jwt.StandardClaims
}

func CreateToken(oid string) (string, error) {
	userClaims := &UserClaims{
		Id: oid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 3600).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func CreateAdminToken(adminId string, adminType int) (string, error) {
	userClaims := &AdminClaims{
		Id:        adminId,
		AdminType: adminType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 3600).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func CreateKeyToken(keyId string) (string, error) {
	keyClaims := KeyClaims{
		KeyId: keyId,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 3600).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, keyClaims)
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

func ParseAdminToken(adminToken string, claims *AdminClaims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(adminToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("wrong method")
		}
		return []byte(secret), nil
	})
}

func ParseKeyToken(keyToken string, claims *KeyClaims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(keyToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("wrong method")
		}
		return []byte(secret), nil
	})
}
