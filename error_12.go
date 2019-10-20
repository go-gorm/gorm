// +build !go1.13

package gorm

func isError(err, target error) bool {
	return err == target
}
