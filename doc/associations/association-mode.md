# Association Mode

Association Mode contains some helper methods to handle relationship things easily.

```go
// Start Association Mode
var user User
db.Model(&user).Association("Languages")
// `user` is the source, it need to be a valid record (contains primary key)
// `Languages` is source's field name for a relationship.
// If those conditions not matched, will return an error, check it with:
// db.Model(&user).Association("Languages").Error


// Query - Find out all related associations
db.Model(&user).Association("Languages").Find(&languages)


// Append - Append new associations for many2many, has_many, will replace current association for has_one, belongs_to
db.Model(&user).Association("Languages").Append([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Append(Language{Name: "DE"})


// Delete - Remove relationship between source & passed arguments, won't delete those arguments
db.Model(&user).Association("Languages").Delete([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Delete(languageZH, languageEN)


// Replace - Replace current associations with new one
db.Model(&user).Association("Languages").Replace([]Language{languageZH, languageEN})
db.Model(&user).Association("Languages").Replace(Language{Name: "DE"}, languageEN)


// Count - Return the count of current associations
db.Model(&user).Association("Languages").Count()


// Clear - Remove relationship between source & current associations, won't delete those associations
db.Model(&user).Association("Languages").Clear()
```
