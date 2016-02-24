# Query Chain

```go
db.First(&first_article).Count(&total_count).Limit(10).Find(&first_page_articles).Offset(10).Find(&second_page_articles)
//// SELECT * FROM articles LIMIT 1; (first_article)
//// SELECT count(*) FROM articles; (total_count)
//// SELECT * FROM articles LIMIT 10; (first_page_articles)
//// SELECT * FROM articles LIMIT 10 OFFSET 10; (second_page_articles)


db.Where("created_at > ?", "2013-10-10").Find(&cancelled_orders, "state = ?", "cancelled").Find(&shipped_orders, "state = ?", "shipped")
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'cancelled'; (cancelled_orders)
//// SELECT * FROM orders WHERE created_at > '2013/10/10' AND state = 'shipped'; (shipped_orders)


// Use variables to keep query chain
todays_orders := db.Where("created_at > ?", "2013-10-29")
cancelled_orders := todays_orders.Where("state = ?", "cancelled")
shipped_orders := todays_orders.Where("state = ?", "shipped")


// Search with shared conditions for different tables
db.Where("product_name = ?", "fancy_product").Find(&orders).Find(&shopping_carts)
//// SELECT * FROM orders WHERE product_name = 'fancy_product'; (orders)
//// SELECT * FROM carts WHERE product_name = 'fancy_product'; (shopping_carts)


// Search with shared conditions from different tables with specified table
db.Where("mail_type = ?", "TEXT").Find(&users1).Table("deleted_users").Find(&users2)
//// SELECT * FROM users WHERE mail_type = 'TEXT'; (users1)
//// SELECT * FROM deleted_users WHERE mail_type = 'TEXT'; (users2)


// FirstOrCreate example
db.Where("email = ?", "x@example.org").Attrs(User{RegisteredIp: "111.111.111.111"}).FirstOrCreate(&user)
//// SELECT * FROM users WHERE email = 'x@example.org';
//// INSERT INTO "users" (email,registered_ip) VALUES ("x@example.org", "111.111.111.111")  // if record not found
```
