package errtranslator

import "encoding/json"

var mysqlErrCodes = map[string]int{
	"uniqueConstraint": 1062,
}

type MysqlErrTranslator struct{}

type MysqlErr struct {
	Number  int    `json:"Number"`
	Message string `json:"Message"`
}

func (m *MysqlErrTranslator) Translate(err error) error {
	parsedErr, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return err
	}

	var mysqlErr MysqlErr
	unmarshalErr := json.Unmarshal(parsedErr, &mysqlErr)
	if unmarshalErr != nil {
		return err
	}

	if mysqlErr.Number == mysqlErrCodes["uniqueConstraint"] {
		return ErrDuplicatedKey{Code: mysqlErr.Number, Message: mysqlErr.Message}
	}

	return err
}
