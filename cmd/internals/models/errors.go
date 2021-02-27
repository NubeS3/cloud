package models

const (
	DbError = iota
	FsError
	Duplicated
	DocumentNotFound
	OtpInvalid
	TokenInvalid
	InvalidAccessKey
	InvalidBucket
	UidMismatch
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

type RouteError struct {
	Msg     string
	ErrType ErrorType
}

func (r *RouteError) Error() string {
	return r.Msg
}
