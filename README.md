# GORM

Yet Another ORM library for Go, hope sucks less. (created for internal usage, API is breakable)

# TODO
Where("id =/>/</<> ?", string or int64).First(&user) (error)
Where("id in (?)", map[]interface{}).First(&user) (error)
Where("id in (?)", map[]interface{}).Find(&users) (error)
Where(map[string]string{"id": "12", "name": "jinzhu"}).Find(&users) (error)
Order("").Limit(11).Or("").Count().Select("").Not("").Offset(11)

First(&user, primary_key) (error)

Save(&user)
Save(&users)
Delete(&user)
Delete(&users)

Before/After Save/Update/Create/Delete
Where("id in ?", map[]interface{}).FindOrInitialize(&users) (error)
Where("id in ?", map[]interface{}).FindOrCreate(&users) (error)

Sql("ssssss", &users)

SQL log
Auto Migration
Index, Unique
