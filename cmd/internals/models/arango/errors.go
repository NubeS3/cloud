package arango

const (
	DbError = iota
	Duplicated
	DocumentNotFound
	Other
)

type ErrorType int

type ModelError struct {
	msg     string
	errType ErrorType
}

func (m *ModelError) Error() string {
	return m.msg
}
