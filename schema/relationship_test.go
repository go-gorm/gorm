package schema_test

import (
	"sync"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
)

func checkStructRelation(t *testing.T, data interface{}, relations ...Relation) {
	if s, err := schema.Parse(data, &sync.Map{}, schema.NamingStrategy{}); err != nil {
		t.Errorf("Failed to parse schema")
	} else {
		for _, rel := range relations {
			checkSchemaRelation(t, s, rel)
		}
	}
}

func TestBelongsToOverrideForeignKey(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name string
	}

	type User struct {
		gorm.Model
		Profile      Profile `gorm:"ForeignKey:ProfileRefer"`
		ProfileRefer int
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profile", Type: schema.BelongsTo, Schema: "User", FieldSchema: "Profile",
		References: []Reference{{"ID", "Profile", "ProfileRefer", "User", "", false}},
	})
}

func TestBelongsToOverrideReferences(t *testing.T) {
	type Profile struct {
		gorm.Model
		Refer string
		Name  string
	}

	type User struct {
		gorm.Model
		Profile   Profile `gorm:"ForeignKey:ProfileID;References:Refer"`
		ProfileID int
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profile", Type: schema.BelongsTo, Schema: "User", FieldSchema: "Profile",
		References: []Reference{{"Refer", "Profile", "ProfileID", "User", "", false}},
	})
}

func TestHasOneOverrideForeignKey(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profile Profile `gorm:"ForeignKey:UserRefer"`
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profile", Type: schema.HasOne, Schema: "User", FieldSchema: "Profile",
		References: []Reference{{"ID", "User", "UserRefer", "Profile", "", true}},
	})
}

func TestHasOneOverrideReferences(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name   string
		UserID uint
	}

	type User struct {
		gorm.Model
		Refer   string
		Profile Profile `gorm:"ForeignKey:UserID;References:Refer"`
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profile", Type: schema.HasOne, Schema: "User", FieldSchema: "Profile",
		References: []Reference{{"Refer", "User", "UserID", "Profile", "", true}},
	})
}

func TestHasManyOverrideForeignKey(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profile []Profile `gorm:"ForeignKey:UserRefer"`
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profile", Type: schema.HasMany, Schema: "User", FieldSchema: "Profile",
		References: []Reference{{"ID", "User", "UserRefer", "Profile", "", true}},
	})
}

func TestHasManyOverrideReferences(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name   string
		UserID uint
	}

	type User struct {
		gorm.Model
		Refer   string
		Profile []Profile `gorm:"ForeignKey:UserID;References:Refer"`
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profile", Type: schema.HasMany, Schema: "User", FieldSchema: "Profile",
		References: []Reference{{"Refer", "User", "UserID", "Profile", "", true}},
	})
}

func TestMany2ManyOverrideForeignKeyAndReferences(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profiles []Profile `gorm:"many2many:user_profiles;ForeignKey:Refer;JoinForeignKey:UserReferID;References:UserRefer;JoinReferences:ProfileRefer"`
		Refer    uint
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profiles", Type: schema.Many2Many, Schema: "User", FieldSchema: "Profile",
		JoinTable: JoinTable{Name: "user_profiles", Table: "user_profiles"},
		References: []Reference{
			{"Refer", "User", "UserReferID", "user_profiles", "", true},
			{"UserRefer", "Profile", "ProfileRefer", "user_profiles", "", false},
		},
	})
}

func TestMany2ManyOverrideForeignKey(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profiles []Profile `gorm:"many2many:user_profiles;ForeignKey:Refer;References:UserRefer"`
		Refer    uint
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profiles", Type: schema.Many2Many, Schema: "User", FieldSchema: "Profile",
		JoinTable: JoinTable{Name: "user_profiles", Table: "user_profiles"},
		References: []Reference{
			{"Refer", "User", "UserRefer", "user_profiles", "", true},
			{"UserRefer", "Profile", "ProfileUserRefer", "user_profiles", "", false},
		},
	})
}

func TestMany2ManyOverrideJoinForeignKey(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profiles []Profile `gorm:"many2many:user_profiles;JoinForeignKey:UserReferID;JoinReferences:ProfileRefer"`
		Refer    uint
	}

	checkStructRelation(t, &User{}, Relation{
		Name: "Profiles", Type: schema.Many2Many, Schema: "User", FieldSchema: "Profile",
		JoinTable: JoinTable{Name: "user_profiles", Table: "user_profiles"},
		References: []Reference{
			{"ID", "User", "UserReferID", "user_profiles", "", true},
			{"ID", "Profile", "ProfileRefer", "user_profiles", "", false},
		},
	})
}

func TestMany2ManyWithMultiPrimaryKeys(t *testing.T) {
	type Tag struct {
		ID     uint   `gorm:"primary_key"`
		Locale string `gorm:"primary_key"`
		Value  string
	}

	type Blog struct {
		ID         uint   `gorm:"primary_key"`
		Locale     string `gorm:"primary_key"`
		Subject    string
		Body       string
		Tags       []Tag `gorm:"many2many:blog_tags;"`
		SharedTags []Tag `gorm:"many2many:shared_blog_tags;ForeignKey:id;References:id"`
		LocaleTags []Tag `gorm:"many2many:locale_blog_tags;ForeignKey:id,locale;References:id"`
	}

	checkStructRelation(t, &Blog{},
		Relation{
			Name: "Tags", Type: schema.Many2Many, Schema: "Blog", FieldSchema: "Tag",
			JoinTable: JoinTable{Name: "blog_tags", Table: "blog_tags"},
			References: []Reference{
				{"ID", "Blog", "BlogID", "blog_tags", "", true},
				{"Locale", "Blog", "BlogLocale", "blog_tags", "", true},
				{"ID", "Tag", "TagID", "blog_tags", "", false},
				{"Locale", "Tag", "TagLocale", "blog_tags", "", false},
			},
		},
		Relation{
			Name: "SharedTags", Type: schema.Many2Many, Schema: "Blog", FieldSchema: "Tag",
			JoinTable: JoinTable{Name: "shared_blog_tags", Table: "shared_blog_tags"},
			References: []Reference{
				{"ID", "Blog", "BlogID", "shared_blog_tags", "", true},
				{"ID", "Tag", "TagID", "shared_blog_tags", "", false},
			},
		},
		Relation{
			Name: "LocaleTags", Type: schema.Many2Many, Schema: "Blog", FieldSchema: "Tag",
			JoinTable: JoinTable{Name: "locale_blog_tags", Table: "locale_blog_tags"},
			References: []Reference{
				{"ID", "Blog", "BlogID", "locale_blog_tags", "", true},
				{"Locale", "Blog", "BlogLocale", "locale_blog_tags", "", true},
				{"ID", "Tag", "TagID", "locale_blog_tags", "", false},
			},
		},
	)
}
