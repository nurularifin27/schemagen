package dbtype

import (
	"strings"
)

type sqliteMapper struct{}

func (sqliteMapper) goType(column Column, opts Options) (string, []string) {
	if goType, imports, ok := preferConfiguredType(column, opts); ok {
		return goType, imports
	}
	if goType, imports, ok := inferByScanType(column); ok {
		return goType, imports
	}

	switch strings.ToLower(column.DatabaseType) {
	case "integer", "int", "tinyint", "smallint", "mediumint", "bigint", "unsigned big int", "int2", "int8":
		return "int64", nil
	case "real", "double", "double precision", "float":
		return "float64", nil
	case "numeric", "decimal":
		return decimalType(opts), nil
	case "boolean":
		return "bool", nil
	case "text", "character", "varchar", "varying character", "nchar", "native character", "nvarchar", "clob":
		return "string", nil
	case "blob":
		return "[]byte", nil
	case "date", "datetime", "timestamp":
		return "time.Time", []string{`"time"`}
	case "json":
		return jsonType(opts)
	default:
		return "string", nil
	}
}
