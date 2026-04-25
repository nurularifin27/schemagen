package dbtype

import (
	"database/sql"
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
		{name: "plain int", column: Column{ScanType: reflect.TypeOf(int(0)), Nullable: false}, expected: "int"},
		{name: "plain uint", column: Column{ScanType: reflect.TypeOf(uint(0)), Nullable: false}, expected: "uint"},
		{name: "null string", column: Column{ScanType: reflect.TypeOf(sql.NullString{}), Nullable: true}, expected: "*string"},
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
		{name: "integer", column: Column{DatabaseType: "integer"}, expected: "int32"},
		{name: "bigint", column: Column{DatabaseType: "bigint"}, expected: "int64"},
		{name: "decimal", column: Column{DatabaseType: "numeric"}, expected: "float64"},
		{name: "datetime", column: Column{DatabaseType: "timestamp"}, expected: "time.Time"},
		{name: "binary", column: Column{DatabaseType: "bytea"}, expected: "[]byte"},
		{name: "jsonb", column: Column{DatabaseType: "jsonb"}, expected: "[]byte"},
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

func TestDriverSpecificFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		driver   string
		column   Column
		expected string
	}{
		{
			name:     "postgres uuid",
			driver:   "postgres",
			column:   Column{DatabaseType: "uuid", FullType: "uuid"},
			expected: "string",
		},
		{
			name:     "postgres text array",
			driver:   "postgres",
			column:   Column{DatabaseType: "_text", FullType: "text[]"},
			expected: "[]string",
		},
		{
			name:     "mysql tinyint bool",
			driver:   "mysql",
			column:   Column{DatabaseType: "tinyint", FullType: "tinyint(1)"},
			expected: "bool",
		},
		{
			name:     "mysql unsigned bigint",
			driver:   "mysql",
			column:   Column{DatabaseType: "bigint", FullType: "bigint unsigned"},
			expected: "uint64",
		},
		{
			name:     "mariadb json",
			driver:   "mariadb",
			column:   Column{DatabaseType: "json", FullType: "json"},
			expected: "[]byte",
		},
		{
			name:     "sqlite integer",
			driver:   "sqlite",
			column:   Column{DatabaseType: "integer", FullType: "integer"},
			expected: "int64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := New(tt.driver).Map(tt.column, "Field")
			if field.GoType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
			}
		})
	}
}

func TestDecimalStrategyString(t *testing.T) {
	field := New("postgres", "string").Map(Column{
		DatabaseType: "numeric",
		FullType:     "numeric(12,2)",
	}, "Amount")
	if field.GoType != "string" {
		t.Fatalf("expected string, got %s", field.GoType)
	}
}

func TestJSONStrategyRawMessage(t *testing.T) {
	field := New("postgres", "", "rawmessage").Map(Column{
		DatabaseType: "jsonb",
		FullType:     "jsonb",
	}, "Payload")
	if field.GoType != "json.RawMessage" {
		t.Fatalf("expected json.RawMessage, got %s", field.GoType)
	}
	if len(field.Imports) != 1 || field.Imports[0] != `"encoding/json"` {
		t.Fatalf("expected encoding/json import, got %#v", field.Imports)
	}
}

func TestNullableStrategySQLNull(t *testing.T) {
	field := New("postgres", "", "", "sqlnull").Map(Column{
		DatabaseType: "timestamp",
		FullType:     "timestamp",
		Nullable:     true,
	}, "CreatedAt")
	if field.GoType != "sql.NullTime" {
		t.Fatalf("expected sql.NullTime, got %s", field.GoType)
	}
	if len(field.Imports) != 1 || field.Imports[0] != `"database/sql"` {
		t.Fatalf("expected imports to include only database/sql, got %#v", field.Imports)
	}
}

func TestOverrideByColumnWinsOverDBType(t *testing.T) {
	field := New("postgres", "float64", "bytes", "", []Override{
		{DBType: "jsonb", GoType: "[]byte"},
		{Table: "users", Column: "payload", GoType: "json.RawMessage", Imports: []string{"encoding/json"}},
	}).Map(Column{
		TableName:    "users",
		Name:         "payload",
		DatabaseType: "jsonb",
		FullType:     "jsonb",
	}, "Payload")
	if field.GoType != "json.RawMessage" {
		t.Fatalf("expected json.RawMessage, got %s", field.GoType)
	}
	if len(field.Imports) != 1 || field.Imports[0] != `"encoding/json"` {
		t.Fatalf("unexpected imports: %#v", field.Imports)
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
