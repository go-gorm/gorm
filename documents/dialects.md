# Dialects

<!-- toc -->

## Dialect Specific Data Type

Certain dialects of SQL ship with their own custom, non-standard column types, such as the `jsonb` column in PostgreSQL. GORM supports loading several of such types, as listed in the following sections.

#### PostgreSQL

GORM supports loading the following PostgreSQL exclusive column types:
- jsonb
- hstore

Given the following Model definition:

```go
import (
	"encoding/json"
	"fmt"
	"reflect"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
)

type Document struct {
	Metadata    postgres.Jsonb
	Secrets     postgres.Hstore
	Body        string

	ID          int
}
```

You may use the model like so:

```go
password := "0654857340"
metadata := json.RawMessage(`{"is_archived": 0}`)
sampleDoc := Document{
    Body : "This is a test document",
    Metadata : postgres.Jsonb{ metadata },
    Secrets : postgres.Hstore{"password" : &password },
}

//insert sampleDoc into the database
db.Create(&sampleDoc)

//retrieve the fields again to confirm if they were inserted correctly
resultDoc := Document{}
db.Where("id = ?", sampleDoc.ID).First(&resultDoc)

metadataIsEqual := reflect.DeepEqual( resultDoc.Metadata, sampleDoc.Metadata)
secretsIsEqual := reflect.DeepEqual( resultDoc.Secrets, resultDoc.Secrets)

//this should print "true"
fmt.Println("Inserted fields are as expected:", metadataIsEqual && secretsIsEqual)

```
