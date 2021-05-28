package ultis

import (
	"regexp"
	"unicode"
)

const (
	EmailPattern      = `^[^\s@]+@[^\s@]+$`
	UsernamePattern   = `^[a-zA-Z][a-zA-Z0-9_\.]{7,24}$`
	BucketNamePattern = `^[a-zA-Z_]*[a-zA-Z0-9\-]{4,64}$`
	FolderNamePattern = `^[a-zA-Z_]*[a-zA-Z0-9\-]{1,32}$`
	FileNamePattern   = `^[0-9a-zA-Z_\-. ]{1,255}$`
)

func ValidateEmail(email string) (bool, error) {
	return regexp.Match(EmailPattern, []byte(email))
}

func ValidateUsername(username string) (bool, error) {
	return regexp.Match(UsernamePattern, []byte(username))
}

func ValidatePassword(password string) (bool, error) {
	upper := false
	lower := false
	number := false
	special := false
	noSpace := true
	for _, c := range password {
		switch {
		case unicode.IsNumber(c):
			number = true
		case unicode.IsUpper(c):
			upper = true
		case unicode.IsLower(c):
			lower = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			special = true
		case unicode.IsSpace(c):
			noSpace = false
		default:
			//return false, false, false, false
		}
	}

	validLength := len(password) >= 8 && len(password) <= 24
	return upper && lower && special && number && validLength && noSpace, nil
}

func ValidateBucketName(name string) (bool, error) {
	return regexp.Match(BucketNamePattern, []byte(name))
}

func ValidateFolderName(name string) (bool, error) {
	return regexp.Match(FolderNamePattern, []byte(name))
}

func ValidateFileName(name string) (bool, error) {
	return regexp.Match(FileNamePattern, []byte(name))
}
