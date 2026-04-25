package dbtype

import "strings"

type mysqlMapper struct{}

func (mysqlMapper) goType(column Column, opts Options) (string, []string) {
	if goType, imports, ok := preferConfiguredType(column, opts); ok {
		return goType, imports
	}
	if goType, imports, ok := inferByScanType(column); ok {
		return goType, imports
	}

	fullType := strings.ToLower(column.FullType)
	switch strings.ToLower(column.DatabaseType) {
	case "tinyint":
		if strings.Contains(fullType, "(1)") {
			return "bool", nil
		}
		if hasUnsigned(fullType) {
			return "uint8", nil
		}
		return "int8", nil
	case "smallint":
		if hasUnsigned(fullType) {
			return "uint16", nil
		}
		return "int16", nil
	case "mediumint", "int", "integer":
		if hasUnsigned(fullType) {
			return "uint32", nil
		}
		return "int32", nil
	case "bigint":
		if hasUnsigned(fullType) {
			return "uint64", nil
		}
		return "int64", nil
	case "decimal", "numeric":
		return decimalType(opts), nil
	case "float":
		return "float32", nil
	case "double", "double precision", "real":
		return "float64", nil
	case "bit":
		return "[]byte", nil
	case "bool", "boolean":
		return "bool", nil
	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext", "enum", "set":
		return "string", nil
	case "date", "datetime", "timestamp":
		return "time.Time", []string{`"time"`}
	case "time":
		return "string", nil
	case "year":
		return "int16", nil
	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		return "[]byte", nil
	case "json":
		return jsonType(opts)
	case "geometry", "point", "linestring", "polygon", "multipoint", "multilinestring", "multipolygon", "geometrycollection":
		return "[]byte", nil
	default:
		return commonTypeFallback(column, opts)
	}
}
