// Package schema turns Go model types and their gorm tags into metadata used by
// GORM statements, callbacks, associations, and migrators. Parsing is local
// reflection work: it neither connects to a database nor confirms the live
// database schema. Application code normally reaches this package through GORM;
// plugins, custom types, and dialectors use its exported metadata and interfaces.
//
// # Parsing and cache ownership
//
// Parse accepts a struct model or pointer/collection wrappers around one. Supply
// a non-nil *sync.Map and Namer, check the returned error, and reuse the cache for
// models with the same stable naming rules:
//
//	cache := &sync.Map{}
//	modelSchema, err := schema.Parse(&User{}, cache, schema.NamingStrategy{})
//	if err != nil {
//		return err
//	}
//	id := modelSchema.LookUpField("ID")
//	if id == nil {
//		return errors.New("User has no ID field")
//	}
//
// Cache entries are keyed by model type, or type plus a special table name.
// Naming strategies are not part of the key. Parse waits for another goroutine
// to finish constructing an existing entry; internal relationship parsing can
// reference an in-progress entry to resolve cycles. Returned Schemas and Fields
// are shared metadata. Treat them as read-only after initialization rather than
// changing tags, names, or relationships while queries run.
//
// A model's Tabler or TablerWithNamer implementation is evaluated on a newly
// allocated model and cached. It cannot choose a table from an individual row.
// Use an operation's Table method or scope for dynamic table routing, or
// ParseWithSpecialTableName when building metadata for an explicit table.
//
// # Model fields and tags
//
// Only exported fields are parsed. Anonymous embedded structs and fields tagged
// embedded are flattened; embeddedPrefix changes their column names while
// BindNames preserves the Go field path. Basic scalar types, time.Time, byte
// collections, custom Scanner/Valuer types, and serializers can represent
// columns. Other eligible struct or slice fields are interpreted as associations
// and must have resolvable key mappings.
//
// Tags are separated by semicolons and their keys are case-insensitive. Common
// controls include column, primaryKey, type, size, default, serializer, and the
// index/constraint tags. The ID field is the conventional primary key when no
// primary key is explicitly selected. Composite keys remain in PrimaryFields;
// PrioritizedPrimaryField is only the preferred field for single-key behavior.
//
// Fields normally permit reads, creates, and updates. Tags can narrow these:
// <-:create permits creation, <-:update permits updates, and -> makes a field
// read-only. ->:false alone disables reading, creation, and updating. Combine it
// with <- to enable writing without reading, for example
// gorm:"->:false;<-:create" for a creation-only field. A bare - disables model
// reads/writes, -:migration skips migration, and -:all disables both. Statement
// Select/Omit and operation-specific rules further restrict fields; Field.Set
// itself does not check these permissions.
//
// HasDefaultValue distinguishes a default from an ordinary zero value.
// DefaultValueInterface holds a parsed Go literal; database expressions, NULL,
// and generated keys can leave it nil. Create callbacks decide whether to supply
// a value or let the database generate one. CreatedAt/UpdatedAt conventions and
// autoCreateTime/autoUpdateTime tags select timestamp handling; integer fields
// can use seconds, milliseconds, or nanoseconds. Parsing records these policies,
// while callbacks apply them when executing the operation.
//
// # Accessors, serializers, and relationships
//
// Field.ValueOf reads a model field and reports Go zero-value status; nil
// embedded pointers read as zero. ReflectValueOf and Set can allocate embedded
// pointers, so use addressable model values of the expected type. NewValuePool
// supplies temporary scan destinations: return borrowed values after assignment
// and do not retain them. Conversion and NULL handling depend on the field type
// or codec; check assignment errors.
//
// A custom scalar type can implement database/sql.Scanner and
// database/sql/driver.Valuer. GormDataTypeInterface supplies a logical type name.
// SerializerInterface provides context-aware field encoding and decoding;
// RegisterSerializer adds a codec selected by a serializer tag. Built-ins are
// json, gob, and unixtime. Register codecs before model parsing and account for
// shared or shallow-copied prototype state. Field-provided CreateClauses,
// QueryClauses, UpdateClauses, and DeleteClauses support behaviors such as soft
// deletion without making those behaviors intrinsic to every model.
//
// Relationship.References records the actual key direction: belongs-to stores
// the foreign key on the owner; has-one/has-many store it on the associated
// model; many-to-many uses a join table. foreignKey and references override
// inferred keys, while joinForeignKey and joinReferences name join-table fields.
// Polymorphic relations add a fixed discriminator. Query-condition helpers use
// already available model identities; they do not fetch missing rows.
//
// # Migration metadata and SQL trust
//
// ParseIndexes, ParseCheckConstraints, ParseUniqueConstraints, and
// Relationship.ParseConstraint describe requested DDL. They do not inspect or
// migrate the database. Identifier quoting is deferred to clause builders and
// dialectors. Expressions in type/default/check/index tags and foreign-key
// actions are trusted schema SQL, not placeholders for untrusted request values.
// Database support for types, indexes, constraints, and referential actions
// remains dialect-specific.
//
// Further examples are in the model, convention, custom-type, serializer,
// association, and index guides:
//
//   - https://gorm.io/docs/models.html
//   - https://gorm.io/docs/conventions.html
//   - https://gorm.io/docs/data_types.html
//   - https://gorm.io/docs/serializer.html
//   - https://gorm.io/docs/associations.html
//   - https://gorm.io/docs/indexes.html
package schema
