package gorm

import "gorm.io/gorm/errtranslator"

func TranslateErr(dialect string, err error) error {
	var errTranslator errtranslator.ErrTranslator

	switch dialect {
	case "sqlite":
		errTranslator = &errtranslator.SqliteErrTranslator{}
	case "postgres":
		errTranslator = &errtranslator.PostgresErrTranslator{}
	case "mysql":
		errTranslator = &errtranslator.MysqlErrTranslator{}
	case "mssql":
		errTranslator = &errtranslator.MssqlErrTranslator{}
	}

	if errTranslator != nil {
		translatedErr := errTranslator.Translate(err)
		if _, ok := translatedErr.(errtranslator.ErrDuplicatedKey); ok {
			return ErrDuplicatedKey
		}
	}

	return err
}
