package schema

import (
	"testing"
)

func TestToDBName(t *testing.T) {
	var maps = map[string]string{
		"":                          "",
		"x":                         "x",
		"X":                         "x",
		"userRestrictions":          "user_restrictions",
		"ThisIsATest":               "this_is_a_test",
		"PFAndESI":                  "pf_and_esi",
		"AbcAndJkl":                 "abc_and_jkl",
		"EmployeeID":                "employee_id",
		"SKU_ID":                    "sku_id",
		"FieldX":                    "field_x",
		"HTTPAndSMTP":               "http_and_smtp",
		"HTTPServerHandlerForURLID": "http_server_handler_for_url_id",
		"UUID":                      "uuid",
		"HTTPURL":                   "http_url",
		"HTTP_URL":                  "http_url",
		"SHA256Hash":                "sha256_hash",
		"SHA256HASH":                "sha256_hash",
		"ThisIsActuallyATestSoWeMayBeAbleToUseThisCodeInGormPackageAlsoIdCanBeUsedAtTheEndAsID": "this_is_actually_a_test_so_we_may_be_able_to_use_this_code_in_gorm_package_also_id_can_be_used_at_the_end_as_id",
	}

	for key, value := range maps {
		if toDBName(key) != value {
			t.Errorf("%v toName should equal %v, but got %v", key, value, toDBName(key))
		}
	}
}

func TestNamingStrategy(t *testing.T) {
	var ns = NamingStrategy{
		TablePrefix:   "public.",
		SingularTable: true,
	}
	idxName := ns.IndexName("public.table", "name")

	if idxName != "idx_public_table_name" {
		t.Errorf("invalid index name generated, got %v", idxName)
	}

	chkName := ns.CheckerName("public.table", "name")
	if chkName != "chk_public_table_name" {
		t.Errorf("invalid checker name generated, got %v", chkName)
	}

	joinTable := ns.JoinTableName("user_languages")
	if joinTable != "public.user_languages" {
		t.Errorf("invalid join table generated, got %v", joinTable)
	}

	joinTable2 := ns.JoinTableName("UserLanguage")
	if joinTable2 != "public.user_language" {
		t.Errorf("invalid join table generated, got %v", joinTable2)
	}

	tableName := ns.TableName("Company")
	if tableName != "public.company" {
		t.Errorf("invalid table name generated, got %v", tableName)
	}
}
