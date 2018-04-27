# AuOrm
基于gorm 实现对原生查询的[]map[string]string{}返回

- 支持增删改查
- 支持事物操作

Installation
------------

Use go get.

	go get github.com/Jetereting/gorm

Then import the validator package into your own code.

	import "github.com/Jetereting/gorm"

示例:
```golang
datas, e := DB.RawMap("select * from users where user_id=?", 123)
if e != nil {
	fmt.Println("err:", e)
}
fmt.Println("datas:", datas)
```

更多示例参照: [Au-ORM 测试](https://github.com/Jetereting/gorm/master/gorm_test.go)

