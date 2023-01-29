package errtranslator

import "encoding/json"

var mssqlErrCodes = map[string]int{
	"uniqueConstraint": 2627,
}

type MssqlErrTranslator struct{}

type MssqlErr struct {
	Number  int    `json:"Number"`
	Message string `json:"Message"`
}

func (m *MssqlErrTranslator) Translate(err error) error {
	parsedErr, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return err
	}

	var mssqlErr MssqlErr
	unmarshalErr := json.Unmarshal(parsedErr, &mssqlErr)
	if unmarshalErr != nil {
		return err
	}

	if mssqlErr.Number == mssqlErrCodes["uniqueConstraint"] {
		return ErrDuplicatedKey{Code: mssqlErr.Number, Message: mssqlErr.Message}
	}

	return err
}
