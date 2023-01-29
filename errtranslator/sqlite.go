package errtranslator

import "encoding/json"

var sqliteErrCodes = map[string]int{
	"uniqueConstraint": 2067,
}

type SqliteErrTranslator struct{}

type SqliteErr struct {
	Code         int `json:"Code"`
	ExtendedCode int `json:"ExtendedCode"`
	SystemErrno  int `json:"SystemErrno"`
}

func (s *SqliteErrTranslator) Translate(err error) error {
	parsedErr, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return err
	}

	var sqliteErr SqliteErr
	unmarshalErr := json.Unmarshal(parsedErr, &sqliteErr)
	if unmarshalErr != nil {
		return err
	}

	if sqliteErr.ExtendedCode == sqliteErrCodes["uniqueConstraint"] {
		return ErrDuplicatedKey{Code: sqliteErr.ExtendedCode, Message: ""}
	}

	return err
}
