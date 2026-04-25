package dbtype

import "strings"

type postgresMapper struct{}

func (postgresMapper) goType(column Column, opts Options) (string, []string) {
	if goType, imports, ok := preferConfiguredType(column, opts); ok {
		return goType, imports
	}
	if goType, imports, ok := inferByScanType(column); ok {
		return goType, imports
	}

	switch strings.ToLower(column.DatabaseType) {
	case "smallint", "int2":
		return "int16", nil
	case "integer", "int", "int4", "serial", "serial4":
		return "int32", nil
	case "bigint", "int8", "bigserial", "serial8":
		return "int64", nil
	case "real", "float4":
		return "float32", nil
	case "double precision", "float8":
		return "float64", nil
	case "numeric", "decimal":
		return decimalType(opts), nil
	case "boolean", "bool":
		return "bool", nil
	case "char", "character", "bpchar", "varchar", "character varying", "text", "citext", "name":
		return "string", nil
	case "date", "time", "time without time zone", "timetz", "time with time zone", "timestamp", "timestamp without time zone", "timestamptz", "timestamp with time zone":
		return "time.Time", []string{`"time"`}
	case "interval":
		return "string", nil
	case "bytea":
		return "[]byte", nil
	case "json", "jsonb":
		return jsonType(opts)
	case "uuid":
		return "string", nil
	case "inet", "cidr", "macaddr", "macaddr8", "xml", "money", "tsvector", "tsquery":
		return "string", nil
	}

	if strings.HasPrefix(column.DatabaseType, "_") {
		return postgresArrayType(column.DatabaseType), nil
	}
	return commonTypeFallback(column, opts)
}

func postgresArrayType(databaseType string) string {
	switch strings.ToLower(databaseType) {
	case "_text", "_varchar", "_bpchar", "_citext", "_uuid":
		return "[]string"
	case "_bool":
		return "[]bool"
	case "_int2":
		return "[]int16"
	case "_int4":
		return "[]int32"
	case "_int8":
		return "[]int64"
	case "_float4":
		return "[]float32"
	case "_float8", "_numeric":
		return "[]float64"
	case "_bytea":
		return "[][]byte"
	default:
		return "[]string"
	}
}
