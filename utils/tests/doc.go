// Package tests provides shared fixtures and assertions for GORM's own test
// suites. Despite its importable path, it is test support rather than an example
// application model layer or a database driver.
//
// The model graph covers belongs-to, has-one, has-many, polymorphic, many-to-many,
// and self-referential associations. Several models intentionally exercise
// nullable keys, nonstandard key types, or composite keys. Their tags describe
// test coverage and should not be copied wholesale as production schema advice.
// Most fixtures embed gorm.Model and therefore include timestamps and soft-delete
// state; Company, Language, Coupon, and CouponProduct use explicit fields instead.
//
// DummyDialector provides callback registration, placeholder rendering, quoting,
// and configurable error translation for isolated statement tests. It does not
// open a database connection, provide migration support, or implement a real
// backend type system. Use a real dialector and an isolated database for tests
// that need execution, migration, or dialect compatibility evidence.
//
// AssertObjEqual compares selected accessible fields; AssertEqual uses permissive
// fixture comparison, including time tolerance, value conversion, and recursive
// slice/struct checks. These helpers assume compatible fixture shapes and are not
// a general assertion library or strict type-equality check. Now returns a fresh
// pointer to the current local time and does not provide a deterministic test
// clock. Tests own their databases, cleanup, and any synchronization of fixtures.
package tests
