package dbtype

import "testing"

func TestCanonicalMappings(t *testing.T) {
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{name: "jsonb", column: Column{DatabaseType: "jsonb"}, expected: "datatypes.JSON"},
		{name: "uuid", column: Column{DatabaseType: "uuid"}, expected: "uuid.UUID"},
		{name: "numeric", column: Column{DatabaseType: "numeric"}, expected: "decimal.Decimal"},
		{name: "date", column: Column{DatabaseType: "date"}, expected: "datatypes.Date"},
		{name: "time", column: Column{DatabaseType: "time"}, expected: "datatypes.Time"},
		{name: "timestamp", column: Column{DatabaseType: "timestamp"}, expected: "time.Time"},
		{name: "tinyint bool", column: Column{DatabaseType: "tinyint", FullType: "tinyint(1)"}, expected: "bool"},
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

func TestArrayMappings(t *testing.T) {
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{name: "text array", column: Column{DatabaseType: "_text", FullType: "text[]"}, expected: "pgtype.FlatArray[string]"},
		{name: "uuid array", column: Column{DatabaseType: "_uuid", FullType: "uuid[]"}, expected: "pgtype.FlatArray[uuid.UUID]"},
		{name: "int8 array", column: Column{DatabaseType: "_int8", FullType: "bigint[]"}, expected: "pgtype.FlatArray[int64]"},
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
