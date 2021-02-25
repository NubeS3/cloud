package models

const (
	DbError = iota
	FsError
	Duplicated
	DocumentNotFound
	OtpInvalid
	InvalidAccessKey
	Other
)

type ErrorType int

type ModelError struct {
	Msg     string
	ErrType ErrorType
}

func (m *ModelError) Error() string {
	return m.Msg
}
