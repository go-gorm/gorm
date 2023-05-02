package tests_test

import (
	"testing"

	"gorm.io/gorm/utils/tests"
	. "gorm.io/gorm/utils/tests"
)

func TestInterface(t *testing.T) {
	vehicleWrite := &tests.Vehicle{Meta: &tests.MotorMeta{Power: "electric"}}

	if err := DB.Create(vehicleWrite).Error; err != nil {
		t.Fatalf("fail to create region %v", err)
	}

	vehicleRead := &tests.Vehicle{Meta: &tests.MotorMeta{}}
	if err := DB.Debug().First(vehicleRead, "id = ?", vehicleWrite.ID).Error; err != nil {
		t.Fatalf("fail to find vehicle %v", err)
	} else {
		AssertEqual(t, vehicleWrite.Meta.(*tests.MotorMeta).Power, vehicleRead.Meta.(*tests.MotorMeta).Power)
	}
}
