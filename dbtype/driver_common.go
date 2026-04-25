package dbtype

import "strings"

type defaultMapper struct{}

func (defaultMapper) goType(column Column, opts Options) (string, []string) {
	if goType, imports, ok := preferConfiguredType(column, opts); ok {
		return goType, imports
	}
	if goType, imports, ok := inferByScanType(column); ok {
		return goType, imports
	}
	return commonTypeFallback(column, opts)
}

func commonTypeFallback(column Column, opts Options) (string, []string) {
	switch strings.ToLower(column.DatabaseType) {
	case "bool", "boolean":
		return "bool", nil
	case "tinyint":
		if strings.Contains(column.FullType, "(1)") {
			return "bool", nil
		}
		return "int8", nil
	case "smallint", "int2", "year":
		return "int16", nil
	case "integer", "int", "serial", "serial4", "int4", "mediumint":
		return "int32", nil
	case "bigint", "bigserial", "serial8", "int8":
		return "int64", nil
	case "real", "float", "float4":
		return "float32", nil
	case "double", "double precision", "float8", "numeric", "decimal":
		return decimalType(opts), nil
	case "bytea", "blob", "binary", "varbinary":
		return "[]byte", nil
	case "date", "datetime", "timestamp", "timestamptz":
		return "time.Time", []string{`"time"`}
	case "json", "jsonb":
		return jsonType(opts)
	case "uuid":
		return "string", nil
	default:
		return "string", nil
	}
}

func preferConfiguredType(column Column, opts Options) (string, []string, bool) {
	switch strings.ToLower(column.DatabaseType) {
	case "numeric", "decimal", "json", "jsonb":
		goType, imports := commonTypeFallback(column, opts)
		return goType, imports, true
	default:
		return "", nil, false
	}
}
