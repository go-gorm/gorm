# Callbacks

<!-- toc -->

You could define callback methods to pointer of model struct, it will be called when creating, updating, querying, deleting, if any callback returns an error, gorm will stop future operations and rollback all changes.

### Creating An Object

Available Callbacks for creating

```go
// begin transaction
BeforeSave
BeforeCreate
// save before associations
// update timestamp `CreatedAt`, `UpdatedAt`
// save self
// reload fields that have default value and its value is blank
// save after associations
AfterCreate
AfterSave
// commit or rollback transaction
```

### Updating An Object

Available Callbacks for updating

```go
// begin transaction
BeforeSave
BeforeUpdate
// save before associations
// update timestamp `UpdatedAt`
// save self
// save after associations
AfterUpdate
AfterSave
// commit or rollback transaction
```

### Deleting An Object

Available Callbacks for deleting

```go
// begin transaction
BeforeDelete
// delete self
AfterDelete
// commit or rollback transaction
```

### Querying An Object

Available Callbacks for querying

```go
// load data from database
// Preloading (edger loading)
AfterFind
```

### Callback Examples

```go
func (u *User) BeforeUpdate() (err error) {
	if u.readonly() {
		err = errors.New("read only user")
	}
	return
}

// Rollback the insertion if user's id greater than 1000
func (u *User) AfterCreate() (err error) {
	if (u.Id > 1000) {
		err = errors.New("user id is already greater than 1000")
	}
	return
}
```

Save/Delete operations in gorm are running in transactions, so changes made in that transaction are not visible unless it is commited.
If you want to use those changes in your callbacks, you need to run your SQL in the same transaction. So you need to pass current transaction to callbacks like this:

```go
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	tx.Model(u).Update("role", "admin")
	return
}
```

```go
func (u *User) AfterCreate(scope *gorm.Scope) (err error) {
  scope.DB().Model(u).Update("role", "admin")
	return
}
```
