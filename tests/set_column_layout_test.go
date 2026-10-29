package tests_test

import (
	"testing"

	"gorm.io/gorm"
)

type SetColumnLayoutModel struct {
	ID        uint
	Address   string
	PriceArea string
	Kind      string
}

func (m *SetColumnLayoutModel) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("price_area", "hook")
	return nil
}

type SetColumnLayoutUpdate struct {
	PriceArea string
	Kind      string
	Address   string
}

type SetColumnLayoutEmbeddedUpdate struct{ SetColumnLayoutUpdate }

type setColumnLayoutTaggedUpdate struct {
	Zone    string `gorm:"column:price_area"`
	Kind    string
	Address string
}

func TestSetColumnDifferentUpdateLayout(t *testing.T) {
	if err := DB.Migrator().DropTable(&SetColumnLayoutModel{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&SetColumnLayoutModel{}); err != nil {
		t.Fatal(err)
	}
	update := SetColumnLayoutUpdate{Address: "new", Kind: "keep", PriceArea: "requested"}
	cases := []struct {
		name    string
		payload interface{}
	}{
		{"pointer", &update},
		{"value", update},
		{"embedded", &SetColumnLayoutEmbeddedUpdate{update}},
		{"column_tag", &setColumnLayoutTaggedUpdate{Zone: "requested", Kind: "keep", Address: "new"}},
		{"same_type", &SetColumnLayoutModel{Address: "new", PriceArea: "requested", Kind: "keep"}},
		{"map", map[string]interface{}{"address": "new", "kind": "keep", "price_area": "requested"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			checkSetColumnLayout(t, test.payload)
		})
	}
	if update.Address != "new" || update.Kind != "keep" || update.PriceArea != "hook" {
		t.Fatalf("wrong payload mutation: %+v", update)
	}
}

func checkSetColumnLayout(t *testing.T, payload interface{}) {
	t.Helper()
	original := SetColumnLayoutModel{Address: "original", PriceArea: "original", Kind: "original"}
	if err := DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Model(&original).Updates(payload).Error; err != nil {
		t.Fatal(err)
	}
	var got SetColumnLayoutModel
	if err := DB.First(&got, original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Address != "new" || got.Kind != "keep" || got.PriceArea != "hook" {
		t.Fatalf("wrong updated columns: %+v; payload: %+v", got, payload)
	}
}
