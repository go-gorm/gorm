package errtranslator

import "fmt"

type ErrTranslator interface {
	Translate(err error) error
}

type ErrDuplicatedKey struct {
	Code    interface{}
	Message string
}

func (e ErrDuplicatedKey) Error() string {
	return fmt.Sprintf("duplicated key not allowed, code: %v, message: %s", e.Code, e.Message)
}
