package schema

import (
	"errors"
	"reflect"
	"strings"
)

// Relationship described the relationship between models
type Relationship struct {
	Kind                           string
	PolymorphicType                string
	PolymorphicDBName              string
	PolymorphicValue               string
	ForeignKey                     []string
	AssociationForeignKey          []string
	JointableForeignkey            []string
	AssociationJointableForeignkey []string
}

func buildToOneRel(field *Field, sourceSchema *Schema) {
	var (
		// user has one profile, associationType is User, profile use UserID as foreign key
		// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
		relationship                              = &Relationship{}
		associationType                           = sourceSchema.ModelType.Name()
		destSchema                                = ParseSchema(reflect.New(field.StructField.Type).Interface())
		tagForeignKeys, tagAssociationForeignKeys []string
	)

	if val := field.TagSettings["REL"]; val != "" {
		relationship.Kind = strings.ToLower(strings.TrimSpace(val))
	}

	if val := field.TagSettings["FOREIGNKEY"]; val != "" {
		tagForeignKeys = toColumns(val)
	}

	if val := field.TagSettings["ASSOCIATION_FOREIGNKEY"]; val != "" {
		tagAssociationForeignKeys = toColumns(val)
	}

	// Cat has one Toy, its `Polymorphic` is `Owner`, then the associationType should be Owner
	// Toy will use field `OwnerID`, `OwnerType` ('cats') as foreign key
	if val := field.TagSettings["POLYMORPHIC"]; val != "" {
		if polymorphicTypeField := getSchemaField(val+"Type", destSchema.Fields); polymorphicTypeField != nil {
			associationType = val
			relationship.PolymorphicType = polymorphicTypeField.Name
			relationship.PolymorphicDBName = polymorphicTypeField.DBName
			// if Cat has several different types of toys set name for each (instead of default 'cats')
			if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
				relationship.PolymorphicValue = value
			} else {
				relationship.PolymorphicValue = ToDBName(sourceSchema.ModelType.Name())
			}
			polymorphicTypeField.IsForeignKey = true
		}
	}

	// Has One
	if (relationship.Kind == "") || (relationship.Kind == "has_one") {
		foreignKeys := tagForeignKeys
		associationForeignKeys := tagAssociationForeignKeys

		// if no foreign keys defined with tag
		if len(foreignKeys) == 0 {
			// if no association foreign keys defined with tag
			if len(associationForeignKeys) == 0 {
				for _, primaryField := range sourceSchema.PrimaryFields {
					foreignKeys = append(foreignKeys, associationType+primaryField.Name)
					associationForeignKeys = append(associationForeignKeys, primaryField.Name)
				}
			} else {
				// generate foreign keys form association foreign keys
				for _, associationForeignKey := range tagAssociationForeignKeys {
					if foreignField := getSchemaField(associationForeignKey, sourceSchema.Fields); foreignField != nil {
						foreignKeys = append(foreignKeys, associationType+foreignField.Name)
						associationForeignKeys = append(associationForeignKeys, foreignField.Name)
					}
				}
			}
		} else {
			// generate association foreign keys from foreign keys
			if len(associationForeignKeys) == 0 {
				for _, foreignKey := range foreignKeys {
					if strings.HasPrefix(foreignKey, associationType) {
						associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
						if foreignField := getSchemaField(associationForeignKey, sourceSchema.Fields); foreignField != nil {
							associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						}
					}
				}
				if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
					associationForeignKeys = []string{getPrimaryPrimaryField(sourceSchema.PrimaryFields).DBName}
				}
			} else if len(foreignKeys) != len(associationForeignKeys) {
				sourceSchema.ParseErrors = append(sourceSchema.ParseErrors, errors.New("invalid foreign keys, should have same length"))
			}
		}

		for idx, foreignKey := range foreignKeys {
			if foreignField := getSchemaField(foreignKey, destSchema.Fields); foreignField != nil {
				if sourceField := getSchemaField(associationForeignKeys[idx], sourceSchema.Fields); sourceField != nil {
					foreignField.IsForeignKey = true
					// source foreign keys
					relationship.AssociationForeignKey = append(relationship.AssociationForeignKey, sourceField.DBName)

					// association foreign keys
					relationship.ForeignKey = append(relationship.ForeignKey, foreignField.DBName)
				}
			}
		}

		if len(relationship.ForeignKey) != 0 {
			relationship.Kind = "has_one"
			field.Relationship = relationship
			return
		}
	}

	// Belongs To
	if (relationship.Kind == "") || (relationship.Kind == "belongs_to") {
		foreignKeys := tagForeignKeys
		associationForeignKeys := tagAssociationForeignKeys

		if len(foreignKeys) == 0 {
			// generate foreign keys & association foreign keys
			if len(associationForeignKeys) == 0 {
				for _, primaryField := range destSchema.PrimaryFields {
					foreignKeys = append(foreignKeys, field.Name+primaryField.Name)
					associationForeignKeys = append(associationForeignKeys, primaryField.Name)
				}
			} else {
				// generate foreign keys with association foreign keys
				for _, associationForeignKey := range associationForeignKeys {
					if foreignField := getSchemaField(associationForeignKey, destSchema.Fields); foreignField != nil {
						foreignKeys = append(foreignKeys, field.Name+foreignField.Name)
						associationForeignKeys = append(associationForeignKeys, foreignField.Name)
					}
				}
			}
		} else {
			// generate foreign keys & association foreign keys
			if len(associationForeignKeys) == 0 {
				for _, foreignKey := range foreignKeys {
					if strings.HasPrefix(foreignKey, field.Name) {
						associationForeignKey := strings.TrimPrefix(foreignKey, field.Name)
						if foreignField := getSchemaField(associationForeignKey, destSchema.Fields); foreignField != nil {
							associationForeignKeys = append(associationForeignKeys, associationForeignKey)
						}
					}
				}
				if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
					associationForeignKeys = []string{getPrimaryPrimaryField(destSchema.PrimaryFields).DBName}
				}
			} else if len(foreignKeys) != len(associationForeignKeys) {
				sourceSchema.ParseErrors = append(sourceSchema.ParseErrors, errors.New("invalid foreign keys, should have same length"))
			}
		}

		for idx, foreignKey := range foreignKeys {
			if foreignField := getSchemaField(foreignKey, sourceSchema.Fields); foreignField != nil {
				if associationField := getSchemaField(associationForeignKeys[idx], destSchema.Fields); associationField != nil {
					foreignField.IsForeignKey = true

					// association foreign keys
					relationship.AssociationForeignKey = append(relationship.AssociationForeignKey, associationField.DBName)

					// source foreign keys
					relationship.ForeignKey = append(relationship.ForeignKey, foreignField.DBName)
				}
			}
		}

		if len(relationship.ForeignKey) != 0 {
			relationship.Kind = "belongs_to"
			field.Relationship = relationship
		}
	}
	return
}

func buildToManyRel(field *Field, sourceSchema *Schema) {
	var (
		relationship                        = &Relationship{}
		elemType                            = field.StructField.Type
		destSchema                          = ParseSchema(reflect.New(elemType).Interface())
		foreignKeys, associationForeignKeys []string
	)

	if val := field.TagSettings["FOREIGNKEY"]; val != "" {
		foreignKeys = toColumns(val)
	}

	if val := field.TagSettings["ASSOCIATION_FOREIGNKEY"]; val != "" {
		associationForeignKeys = toColumns(val)
	}

	for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() == reflect.Struct {
		if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
			relationship.Kind = "many_to_many"

			{ // Foreign Keys for Source
				joinTableDBNames := []string{}

				if val := field.TagSettings["JOINTABLE_FOREIGNKEY"]; val != "" {
					joinTableDBNames = toColumns(val)
				}

				// if no foreign keys defined with tag
				if len(foreignKeys) == 0 {
					for _, field := range sourceSchema.PrimaryFields {
						foreignKeys = append(foreignKeys, field.DBName)
					}
				}

				for idx, foreignKey := range foreignKeys {
					if foreignField := getSchemaField(foreignKey, sourceSchema.Fields); foreignField != nil {
						// source foreign keys (db names)
						relationship.ForeignKey = append(relationship.ForeignKey, foreignField.DBName)

						// setup join table foreign keys for source
						// if defined join table's foreign key with tag
						if len(joinTableDBNames) > idx {
							relationship.JointableForeignkey = append(relationship.JointableForeignkey, joinTableDBNames[idx])
						} else {
							defaultJointableForeignKey := ToDBName(sourceSchema.ModelType.Name()) + "_" + foreignField.DBName
							relationship.JointableForeignkey = append(relationship.JointableForeignkey, defaultJointableForeignKey)
						}
					}
				}
			}

			{ // Foreign Keys for Association (Destination)
				associationJoinTableDBNames := []string{}

				if foreignKey := field.TagSettings["ASSOCIATION_JOINTABLE_FOREIGNKEY"]; foreignKey != "" {
					associationJoinTableDBNames = strings.Split(foreignKey, ",")
				}

				// if no association foreign keys defined with tag
				if len(associationForeignKeys) == 0 {
					for _, field := range destSchema.PrimaryFields {
						associationForeignKeys = append(associationForeignKeys, field.DBName)
					}
				}

				for idx, name := range associationForeignKeys {
					if field := getSchemaField(name, destSchema.Fields); field != nil {
						// association foreign keys (db names)
						relationship.AssociationForeignKey = append(relationship.AssociationForeignKey, field.DBName)

						// setup join table foreign keys for association
						if len(associationJoinTableDBNames) > idx {
							relationship.AssociationJointableForeignkey = append(relationship.AssociationJointableForeignkey, associationJoinTableDBNames[idx])
						} else {
							// join table foreign keys for association
							joinTableDBName := ToDBName(elemType.Name()) + "_" + field.DBName
							relationship.AssociationJointableForeignkey = append(relationship.AssociationJointableForeignkey, joinTableDBName)
						}
					}
				}
			}

			field.Relationship = relationship
		} else {
			// User has many comments, associationType is User, comment use UserID as foreign key
			associationType := sourceSchema.ModelType.Name()
			relationship.Kind = "has_many"

			if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
				// Dog has many toys, tag polymorphic is Owner, then associationType is Owner
				// Toy use OwnerID, OwnerType ('dogs') as foreign key
				if polymorphicType := getSchemaField(polymorphic+"Type", destSchema.Fields); polymorphicType != nil {
					associationType = polymorphic
					relationship.PolymorphicType = polymorphicType.Name
					relationship.PolymorphicDBName = polymorphicType.DBName
					// if Dog has multiple set of toys set name of the set (instead of default 'dogs')
					if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
						relationship.PolymorphicValue = value
					} else {
						relationship.PolymorphicValue = ToDBName(sourceSchema.ModelType.Name())
					}
					polymorphicType.IsForeignKey = true
				}
			}

			// if no foreign keys defined with tag
			if len(foreignKeys) == 0 {
				// if no association foreign keys defined with tag
				if len(associationForeignKeys) == 0 {
					for _, field := range sourceSchema.PrimaryFields {
						foreignKeys = append(foreignKeys, associationType+field.Name)
						associationForeignKeys = append(associationForeignKeys, field.Name)
					}
				} else {
					// generate foreign keys from defined association foreign keys
					for _, scopeFieldName := range associationForeignKeys {
						if foreignField := getSchemaField(scopeFieldName, sourceSchema.Fields); foreignField != nil {
							foreignKeys = append(foreignKeys, associationType+foreignField.Name)
							associationForeignKeys = append(associationForeignKeys, foreignField.Name)
						}
					}
				}
			} else {
				// generate association foreign keys from foreign keys
				if len(associationForeignKeys) == 0 {
					for _, foreignKey := range foreignKeys {
						if strings.HasPrefix(foreignKey, associationType) {
							associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
							if foreignField := getSchemaField(associationForeignKey, sourceSchema.Fields); foreignField != nil {
								associationForeignKeys = append(associationForeignKeys, associationForeignKey)
							}
						}
					}
					if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
						associationForeignKeys = []string{getPrimaryPrimaryField(sourceSchema.PrimaryFields).DBName}
					}
				} else if len(foreignKeys) != len(associationForeignKeys) {
					sourceSchema.ParseErrors = append(sourceSchema.ParseErrors, errors.New("invalid foreign keys, should have same length"))
				}
			}

			for idx, foreignKey := range foreignKeys {
				if foreignField := getSchemaField(foreignKey, destSchema.Fields); foreignField != nil {
					if associationField := getSchemaField(associationForeignKeys[idx], sourceSchema.Fields); associationField != nil {
						// source foreign keys
						foreignField.IsForeignKey = true
						relationship.AssociationForeignKey = append(relationship.AssociationForeignKey, associationField.DBName)

						// association foreign keys
						relationship.ForeignKey = append(relationship.ForeignKey, foreignField.DBName)
					}
				}
			}

			if len(relationship.ForeignKey) != 0 {
				field.Relationship = relationship
			}
		}
	} else {
		field.IsNormal = true
	}
	return
}
