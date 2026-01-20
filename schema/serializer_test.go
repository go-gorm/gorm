package schema

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestUnixSecondSerializer_Value(t *testing.T) {
	var (
		intValue      = math.MaxInt64
		int8Value     = int8(math.MaxInt8)
		int16Value    = int16(math.MaxInt16)
		int32Value    = int32(math.MaxInt32)
		int64Value    = int64(math.MaxInt64)
		uintValue     = uint(math.MaxInt64)
		uint8Value    = uint8(math.MaxUint8)
		uint16Value   = uint16(math.MaxUint16)
		uint32Value   = uint32(math.MaxUint32)
		uint64Value   = uint64(math.MaxInt64)
		maxInt64Plus1 = uint64(math.MaxInt64 + 1)

		intPtrValue      = &intValue
		int8PtrValue     = &int8Value
		int16PtrValue    = &int16Value
		int32PtrValue    = &int32Value
		int64PtrValue    = &int64Value
		uintPtrValue     = &uintValue
		uint8PtrValue    = &uint8Value
		uint16PtrValue   = &uint16Value
		uint32PtrValue   = &uint32Value
		uint64PtrValue   = &uint64Value
		maxInt64Plus1Ptr = &maxInt64Plus1
	)
	tests := []struct {
		name    string
		value   interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:    "int",
			value:   intValue,
			want:    time.Unix(int64(intValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "int8",
			value:   int8Value,
			want:    time.Unix(int64(int8Value), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "int16",
			value:   int16Value,
			want:    time.Unix(int64(int16Value), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "int32",
			value:   int32Value,
			want:    time.Unix(int64(int32Value), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "int64",
			value:   int64Value,
			want:    time.Unix(int64Value, 0).UTC(),
			wantErr: false,
		},
		{
			name:    "uint",
			value:   uintValue,
			want:    time.Unix(int64(uintValue), 0).UTC(), //nolint:gosec
			wantErr: false,
		},
		{
			name:    "uint8",
			value:   uint8Value,
			want:    time.Unix(int64(uint8Value), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "uint16",
			value:   uint16Value,
			want:    time.Unix(int64(uint16Value), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "uint32",
			value:   uint32Value,
			want:    time.Unix(int64(uint32Value), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "uint64",
			value:   uint64Value,
			want:    time.Unix(int64(uint64Value), 0).UTC(), //nolint:gosec
			wantErr: false,
		},
		{
			name:    "maxInt64+1",
			value:   maxInt64Plus1,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "*int",
			value:   intPtrValue,
			want:    time.Unix(int64(*intPtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*int8",
			value:   int8PtrValue,
			want:    time.Unix(int64(*int8PtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*int16",
			value:   int16PtrValue,
			want:    time.Unix(int64(*int16PtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*int32",
			value:   int32PtrValue,
			want:    time.Unix(int64(*int32PtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*int64",
			value:   int64PtrValue,
			want:    time.Unix(*int64PtrValue, 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*uint",
			value:   uintPtrValue,
			want:    time.Unix(int64(*uintPtrValue), 0).UTC(), //nolint:gosec
			wantErr: false,
		},
		{
			name:    "*uint8",
			value:   uint8PtrValue,
			want:    time.Unix(int64(*uint8PtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*uint16",
			value:   uint16PtrValue,
			want:    time.Unix(int64(*uint16PtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*uint32",
			value:   uint32PtrValue,
			want:    time.Unix(int64(*uint32PtrValue), 0).UTC(),
			wantErr: false,
		},
		{
			name:    "*uint64",
			value:   uint64PtrValue,
			want:    time.Unix(int64(*uint64PtrValue), 0).UTC(), //nolint:gosec
			wantErr: false,
		},
		{
			name:    "pointer to maxInt64+1",
			value:   maxInt64Plus1Ptr,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "nil pointer",
			value:   (*int)(nil),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid type",
			value:   "invalid",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnixSecondSerializer{}.Value(context.Background(), nil, reflect.Value{}, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnixSecondSerializer.Value() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if tt.want == nil && got == nil {
				return
			}
			if tt.want == nil {
				t.Fatalf("UnixSecondSerializer.Value() = %v, want nil", got)
			}
			if got == nil {
				t.Fatalf("UnixSecondSerializer.Value() = nil, want %v", tt.want)
			}
			if gotTime, ok := got.(time.Time); !ok {
				t.Errorf("UnixSecondSerializer.Value() returned %T, expected time.Time", got)
			} else if !tt.want.(time.Time).Equal(gotTime) {
				t.Errorf("UnixSecondSerializer.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPgArraySerializer_Scan(t *testing.T) {
	tests := []struct {
		name    string
		dbValue interface{}
		want    []string
		wantErr bool
	}{
		{
			name:    "normal array",
			dbValue: "{foo,bar,baz}",
			want:    []string{"foo", "bar", "baz"},
			wantErr: false,
		},
		{
			name:    "empty array",
			dbValue: "{}",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "single element",
			dbValue: "{single}",
			want:    []string{"single"},
			wantErr: false,
		},
		{
			name:    "quoted element with comma",
			dbValue: `{"hello, world",test}`,
			want:    []string{"hello, world", "test"},
			wantErr: false,
		},
		{
			name:    "escaped quote",
			dbValue: `{"say \"hi\""}`,
			want:    []string{`say "hi"`},
			wantErr: false,
		},
		{
			name:    "escaped backslash",
			dbValue: `{"path\\to"}`,
			want:    []string{`path\to`},
			wantErr: false,
		},
		{
			name:    "invalid format",
			dbValue: "not an array",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, ok := tt.dbValue.(string)
			if !ok {
				t.Skip("skipping non-string test")
			}

			got, err := parsePgArray(str)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePgArray() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("parsePgArray() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parsePgArray()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPgArraySerializer_Value(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:    "normal slice",
			value:   []string{"a", "b", "c"},
			want:    "{a,b,c}",
			wantErr: false,
		},
		{
			name:    "single element",
			value:   []string{"only"},
			want:    "{only}",
			wantErr: false,
		},
		{
			name:    "element with comma",
			value:   []string{"hello, world"},
			want:    `{"hello, world"}`,
			wantErr: false,
		},
		{
			name:    "element with quote",
			value:   []string{`say "hi"`},
			want:    `{"say \"hi\""}`,
			wantErr: false,
		},
		{
			name:    "element with backslash",
			value:   []string{`path\to`},
			want:    `{"path\\to"}`,
			wantErr: false,
		},
		{
			name:    "empty slice",
			value:   []string{},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "nil slice",
			value:   ([]string)(nil),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "nil value",
			value:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "element with space",
			value:   []string{"hello world"},
			want:    `{"hello world"}`,
			wantErr: false,
		},
		{
			name:    "element with braces",
			value:   []string{"{test}"},
			want:    `{"{test}"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PgArraySerializer{}.Value(context.Background(), nil, reflect.Value{}, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("PgArraySerializer.Value() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("PgArraySerializer.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapePgArrayElement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello", "hello"},
		{"with comma", "hello, world", `"hello, world"`},
		{"with quote", `say "hi"`, `"say \"hi\""`},
		{"with backslash", `path\to`, `"path\\to"`},
		{"with space", "hello world", `"hello world"`},
		{"with braces", "{test}", `"{test}"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapePgArrayElement(tt.input)
			if got != tt.want {
				t.Errorf("escapePgArrayElement(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
