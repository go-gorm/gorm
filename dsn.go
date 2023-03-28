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
	DB      string
	Options map[string]string
}

func (d DSN) String() string {
	dsn := d.User + ":" + d.Pass + "@tcp(" + d.Host + ":" + strconv.Itoa(d.Port) + ")/" + d.DB

	if d.Options != nil && len(d.Options) > 0 {
		value := url.Values{}

		for k, v := range d.Options {
			value.Add(k, v)
		}

		dsn += "?" + value.Encode()
	}

	return dsn
}
