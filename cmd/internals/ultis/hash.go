package ultis

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
)

func SHA256(key string) (string, error) {
	h := sha256.New()
	_, err := io.WriteString(h, key)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func SHA1(key string) (string, error) {
	h := sha1.New()
	_, err := io.WriteString(h, key)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func SHA512(key string) (string, error) {
	h := sha512.New()
	_, err := io.WriteString(h, key)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func GetHashFunc(name string) (func(string) (string, error), error) {
	switch name {
	case "SHA1":
		return SHA1, nil
	case "SHA256":
		return SHA256, nil
	case "SHA512":
		return SHA512, nil
	default:
		return nil, errors.New("unknown hash function")
	}
}
