package gorm

import (
	"net/url"
	"strconv"
)

type DSN struct {
	Host    string
	Port    int
	User    string
	Pass    string
	Db      string
	Options map[string]string
}

func (d DSN) String() string {
	dsn := d.User + ":" + d.Pass + "@tcp(" + d.Host + ":" + strconv.Itoa(d.Port) + ")/" + d.Db

	if d.Options != nil {
		value := url.Values{}

		for k, v := range d.Options {
			value.Add(k, v)
		}

		dsn += "?" + value.Encode()
	}

	return dsn
}
