//go:build go1.27

package gorm

import "testing"

func TestDBGenericG(t *testing.T) {
	db := &DB{}
	generic := db.G[struct{}]()
	// Unwrap the public interface to verify that G binds it to the receiver.
	got, ok := generic.(*g[struct{}])
	if !ok {
		t.Fatalf("G returned %T, want *g[struct{}]", generic)
	}
	if got.db != db {
		t.Fatal("G did not bind the generic interface to its receiver")
	}
}
