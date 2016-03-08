package postgres

import (
	"database/sql"
	"database/sql/driver"

	_ "github.com/lib/pq"
	"github.com/lib/pq/hstore"
)

type Hstore map[string]*string

// Value get value of Hstore
func (h Hstore) Value() (driver.Value, error) {
	hstore := hstore.Hstore{Map: map[string]sql.NullString{}}
	if len(h) == 0 {
		return nil, nil
	}

	for key, value := range h {
		var s sql.NullString
		if value != nil {
			s.String = *value
			s.Valid = true
		}
		hstore.Map[key] = s
	}
	return hstore.Value()
}

// Scan scan value into Hstore
func (h *Hstore) Scan(value interface{}) error {
	hstore := hstore.Hstore{}

	if err := hstore.Scan(value); err != nil {
		return err
	}

	if len(hstore.Map) == 0 {
		return nil
	}

	*h = Hstore{}
	for k := range hstore.Map {
		if hstore.Map[k].Valid {
			s := hstore.Map[k].String
			(*h)[k] = &s
		} else {
			(*h)[k] = nil
		}
	}

	return nil
}
