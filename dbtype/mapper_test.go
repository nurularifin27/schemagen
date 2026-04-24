package dbtype

import "testing"

func TestDriverStrategyCanonicalMappings(t *testing.T) {
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
		{name: "tinyint signed", column: Column{DatabaseType: "tinyint", FullType: "tinyint"}, expected: "int8"},
		{name: "tinyint unsigned", column: Column{DatabaseType: "tinyint", FullType: "tinyint unsigned"}, expected: "uint8"},
		{name: "int unsigned", column: Column{DatabaseType: "int", FullType: "int unsigned"}, expected: "uint32"},
		{name: "bigint unsigned", column: Column{DatabaseType: "bigint", FullType: "bigint unsigned"}, expected: "uint64"},
		{name: "bit bytes", column: Column{DatabaseType: "bit", FullType: "bit(8)"}, expected: "[]byte"},
	}

	mapper := New("postgres", StrategyDriver)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mapper.Map(tt.column, "Field")
			if field.GoType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
			}
		})
	}
}

func TestGormStrategyCanonicalMappings(t *testing.T) {
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{name: "jsonb", column: Column{DatabaseType: "jsonb"}, expected: "datatypes.JSON"},
		{name: "uuid", column: Column{DatabaseType: "uuid"}, expected: "string"},
		{name: "numeric", column: Column{DatabaseType: "numeric"}, expected: "float64"},
		{name: "date", column: Column{DatabaseType: "date"}, expected: "time.Time"},
		{name: "time", column: Column{DatabaseType: "time"}, expected: "string"},
		{name: "timestamp", column: Column{DatabaseType: "timestamp"}, expected: "time.Time"},
		{name: "tinyint unsigned", column: Column{DatabaseType: "tinyint", FullType: "tinyint unsigned"}, expected: "uint8"},
	}

	mapper := New("postgres", StrategyGorm)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mapper.Map(tt.column, "Field")
			if field.GoType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
			}
		})
	}
}

func TestNullableSpecialTypes(t *testing.T) {
	tests := []struct {
		name     string
		mapper   Mapper
		column   Column
		expected string
	}{
		{name: "nullable json stays value", mapper: New("postgres", StrategyDriver), column: Column{DatabaseType: "jsonb", Nullable: true}, expected: "datatypes.JSON"},
		{name: "nullable uuid driver pointer", mapper: New("postgres", StrategyDriver), column: Column{DatabaseType: "uuid", Nullable: true}, expected: "*uuid.UUID"},
		{name: "nullable uuid gorm pointer", mapper: New("postgres", StrategyGorm), column: Column{DatabaseType: "uuid", Nullable: true}, expected: "*string"},
		{name: "nullable decimal driver pointer", mapper: New("postgres", StrategyDriver), column: Column{DatabaseType: "numeric", Nullable: true}, expected: "*decimal.Decimal"},
		{name: "nullable decimal gorm pointer", mapper: New("postgres", StrategyGorm), column: Column{DatabaseType: "numeric", Nullable: true}, expected: "*float64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.mapper.Map(tt.column, "Field")
			if field.GoType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
			}
		})
	}
}

func TestArrayMappingsStayDriverAware(t *testing.T) {
	tests := []struct {
		name     string
		column   Column
		expected string
	}{
		{name: "text array", column: Column{DatabaseType: "_text", FullType: "text[]"}, expected: "pgtype.FlatArray[string]"},
		{name: "uuid array", column: Column{DatabaseType: "_uuid", FullType: "uuid[]"}, expected: "pgtype.FlatArray[uuid.UUID]"},
		{name: "int8 array", column: Column{DatabaseType: "_int8", FullType: "bigint[]"}, expected: "pgtype.FlatArray[int64]"},
	}

	for _, strategy := range []string{StrategyDriver, StrategyGorm} {
		mapper := New("postgres", strategy)
		for _, tt := range tests {
			t.Run(strategy+"/"+tt.name, func(t *testing.T) {
				field := mapper.Map(tt.column, "Field")
				if field.GoType != tt.expected {
					t.Fatalf("expected %s, got %s", tt.expected, field.GoType)
				}
			})
		}
	}
}
