package dbtype

import (
	"reflect"
	"testing"
	"time"
)

func TestMapUsesScanTypeInference(t *testing.T) {
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{name: "string", column: Column{ScanType: reflect.TypeOf(""), Nullable: false}, expected: "string"},
		{name: "nullable string", column: Column{ScanType: reflect.TypeOf(""), Nullable: true}, expected: "*string"},
		{name: "int64", column: Column{ScanType: reflect.TypeOf(int64(0)), Nullable: false}, expected: "int64"},
		{name: "nullable int64", column: Column{ScanType: reflect.TypeOf(int64(0)), Nullable: true}, expected: "*int64"},
		{name: "bytes", column: Column{ScanType: reflect.TypeOf([]byte{}), Nullable: true}, expected: "[]byte"},
		{name: "time", column: Column{ScanType: reflect.TypeOf(time.Time{}), Nullable: false}, expected: "time.Time"},
		{name: "nullable time", column: Column{ScanType: reflect.TypeOf(time.Time{}), Nullable: true}, expected: "*time.Time"},
	}

	mapper := New("postgres")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mapper.Map(tt.column, "Field")
			if field.GoType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
			}
		})
	}
}

func TestMapFallsBackWithoutScanType(t *testing.T) {
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{name: "bool", column: Column{DatabaseType: "boolean"}, expected: "bool"},
		{name: "tinyint bool", column: Column{DatabaseType: "tinyint", FullType: "tinyint(1)"}, expected: "bool"},
		{name: "integer", column: Column{DatabaseType: "integer"}, expected: "int32"},
		{name: "bigint", column: Column{DatabaseType: "bigint"}, expected: "int64"},
		{name: "decimal", column: Column{DatabaseType: "numeric"}, expected: "float64"},
		{name: "datetime", column: Column{DatabaseType: "timestamp"}, expected: "time.Time"},
		{name: "binary", column: Column{DatabaseType: "bytea"}, expected: "[]byte"},
		{name: "default string", column: Column{DatabaseType: "jsonb"}, expected: "string"},
	}

	mapper := New("postgres")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mapper.Map(tt.column, "Field")
			if field.GoType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
			}
		})
	}
}

func TestMapDoesNotAddTypeTag(t *testing.T) {
	field := New("postgres").Map(Column{
		Name:         "metadata",
		DatabaseType: "jsonb",
		FullType:     "jsonb",
		ScanType:     reflect.TypeOf([]byte{}),
	}, "Metadata")

	for _, tag := range field.Tags {
		if len(tag) >= 5 && tag[:5] == "type:" {
			t.Fatalf("expected no type tag, got %#v", field.Tags)
		}
	}
}
