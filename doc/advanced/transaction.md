## Transactions

To perform a set of operations within a transaction, the general flow is as below.
The database handle returned from ``` db.Begin() ``` should be used for all operations within the transaction.
(Note that all individual save and delete operations are run in a transaction by default.)

```go
// begin
tx := db.Begin()

// do some database operations (use 'tx' from this point, not 'db')
tx.Create(...)
...

// rollback in case of error
tx.Rollback()

// Or commit if all is ok
tx.Commit()
```

### A Specific Example
```
func CreateAnimals(db *gorm.DB) err {
  tx := db.Begin()
  // Note the use of tx as the database handle once you are within a transaction

  if err := tx.Create(&Animal{Name: "Giraffe"}).Error; err != nil {
     tx.Rollback()
     return err
  }

  if err := tx.Create(&Animal{Name: "Lion"}).Error; err != nil {
     tx.Rollback()
     return err
  }

  tx.Commit()
  return nil
}
```
