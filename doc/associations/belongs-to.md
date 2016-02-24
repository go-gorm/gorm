# Belongs To

```go
// User belongs to a profile, ProfileID is the foreign key
type User struct {
  gorm.Model
  Profile   Profile
  ProfileID int
}

type Profile struct {
  gorm.Model
  Name   string
}

db.Model(&user).Related(&profile)
//// SELECT * FROM profiles WHERE id = 111; // 111 is user's foreign key ProfileID
```
