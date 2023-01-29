package errtranslator

import "encoding/json"

var postgresErrCodes = map[string]string{
	"uniqueConstraint": "23505",
}

type PostgresErrTranslator struct{}

type PostgresErr struct {
	Code     string `json:"Code"`
	Severity string `json:"Severity"`
	Message  string `json:"Message"`
}

func (p *PostgresErrTranslator) Translate(err error) error {
	parsedErr, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return err
	}

	var postgresErr PostgresErr
	unmarshalErr := json.Unmarshal(parsedErr, &postgresErr)
	if unmarshalErr != nil {
		return err
	}

	if postgresErr.Code == postgresErrCodes["uniqueConstraint"] {
		return ErrDuplicatedKey{Code: postgresErr.Code, Message: postgresErr.Message}
	}

	return err
}
