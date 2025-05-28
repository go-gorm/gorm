package apaas

import "fmt"

const MSG_PREFIX = "[apaas_engine]"

func GenError(msg string) error {
	return fmt.Errorf("%s %s", MSG_PREFIX, msg)
}
