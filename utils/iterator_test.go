//go:build go1.23
// +build go1.23

package utils

import (
	"iter"
	"maps"
	"reflect"
	"testing"
)

func TestIsIteratorSeq(t *testing.T) {
	// Create a simple Seq iterator
	m := map[int]any{0: nil, 1: nil, 2: nil}
	seq := maps.Keys(m)

	v := reflect.ValueOf(seq)
	if !isIteratorSeq(v) {
		t.Error("Expected IsIteratorSeq to return true for iter.Seq[int]")
	}

	// Test with non-iterator
	notSeq := func() {}
	v2 := reflect.ValueOf(notSeq)
	if isIteratorSeq(v2) {
		t.Error("Expected IsIteratorSeq to return false for regular function")
	}
}

func TestConvertIteratorSeqToSlice(t *testing.T) {
	// Create test data
	expected := []int{1, 2, 3}

	// Create iterator
	var seq iter.Seq[int] = func(yield func(int) bool) {
		for _, val := range expected {
			if !yield(val) {
				return
			}
		}
	}

	v := reflect.ValueOf(seq)
	result := convertIteratorSeqToSlice(v)

	if result.Kind() != reflect.Slice {
		t.Errorf("Expected slice, got %v", result.Kind())
	}

	resultSlice := result.Interface().([]int)
	if len(resultSlice) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(resultSlice))
	}

	for i, val := range resultSlice {
		if val != expected[i] {
			t.Errorf("Expected %d at index %d, got %d", expected[i], i, val)
		}
	}
}

func TestConvertIteratorToSlice(t *testing.T) {
	// Test with Seq
	seq := func(yield func(string) bool) {
		words := []string{"foo", "bar", "baz"}
		for _, word := range words {
			if !yield(word) {
				return
			}
		}
	}

	v := reflect.ValueOf(seq)
	result, converted := ConvertIteratorToSlice(v)

	if !converted {
		t.Error("Expected conversion to succeed for iter.Seq")
	}

	resultSlice := result.Interface().([]string)
	expected := []string{"foo", "bar", "baz"}

	if len(resultSlice) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(resultSlice))
	}

	// Test with non-iterator
	notIterator := "not an iterator"
	v2 := reflect.ValueOf(notIterator)
	_, converted2 := ConvertIteratorToSlice(v2)

	if converted2 {
		t.Error("Expected conversion to fail for non-iterator")
	}
}
