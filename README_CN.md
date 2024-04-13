# GORM



为 Golang 设计的出色 ORM 库，旨在使开发人员更加友好地开发

[![去报告卡](https://camo.githubusercontent.com/f11e6cf2e617c26212fc0ba1c280effd9838daeaf3f4177566e7ab736a5eec6c/68747470733a2f2f676f7265706f7274636172642e636f6d2f62616467652f6769746875622e636f6d2f676f2d676f726d2f676f726d)](https://goreportcard.com/report/github.com/go-gorm/gorm) [![测试状态](https://github.com/go-gorm/gorm/workflows/tests/badge.svg?branch=master)](https://github.com/go-gorm/gorm/actions) [![麻省理工学院许可证](https://camo.githubusercontent.com/b5241c02eaadcf18b16b74247e96df65b377f820ef4ff4109b77f689ed0963d2/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f6c6963656e73652d4d49542d627269676874677265656e2e737667)](https://opensource.org/licenses/MIT) [![Go.Dev 参考](https://camo.githubusercontent.com/d9c8c216896231065f3aed3efe680fb59e56a8a5d078a6d513ff6d178b512bb1/68747470733a2f2f696d672e736869656c64732e696f2f62616467652f676f2e6465762d7265666572656e63652d626c75653f6c6f676f3d676f266c6f676f436f6c6f723d7768697465)](https://pkg.go.dev/gorm.io/gorm?tab=doc)

[**🌎English Documentation**](./README.md)

## 概述



- 对象关联（Has One、Has Many、Belongs To、Many To Many、Polymorphism、Single-table inheritance）
- 钩子函数（Before/After Create/Save/Update/Delete/Find）
- 预加载和连接查询（Eager loading with Preload, Joins）
- 事务（Transactions）、嵌套事务（Nested Transactions）、保存点（Save Point）、回滚到保存点（RollbackTo to Saved Point）
- 上下文（Context）、预编译语句模式（Prepared Statement Mode）、空运行（DryRun Mode）
- 批量插入（Batch Insert）、批量查找（FindInBatches）、转换为映射（Find To Map）
- SQL 构建器（SQL Builder）、更新/插入（Upsert）、锁定（Locking）、优化器/索引/注释提示（Optimizer/Index/Comment Hints）、命名参数（NamedArg）、带有 SQL 表达式的搜索/更新/创建（Search/Update/Create with SQL Expr）
- 复合主键（Composite Primary Key）
- 自动迁移（Auto Migrations）
- 日志记录器（Logger）
- 可扩展、灵活的插件 API：数据库解析器（Database Resolver，支持多数据库、读写分离）/ Prometheus…
- 每个特性都附带测试
- 开发者友好（Developer Friendly）

## 入门



- GORM 指南 [https://gorm.io](https://gorm.io/)
- 生成指南 https://gorm.io/gen/index.html

## 贡献



[您可以帮助交付更好的 GORM，查看您可以做的事情](https://gorm.io/contribute.html)

## 贡献者



[感谢您](https://github.com/go-gorm/gorm/graphs/contributors)对 GORM 框架的贡献！

## 执照



© Jinzhu, 2013~time.Now

根据[MIT 许可证发布](https://github.com/go-gorm/gorm/blob/master/LICENSE)
