package dbtype

import "strings"

type mariaDBMapper struct{}

func (mariaDBMapper) goType(column Column, opts Options) (string, []string) {
	if goType, imports, ok := preferConfiguredType(column, opts); ok {
		return goType, imports
	}
	if goType, imports, ok := inferByScanType(column); ok {
		return goType, imports
	}

	switch strings.ToLower(column.DatabaseType) {
	case "json":
		return jsonType(opts)
	default:
		return mysqlMapper{}.goType(column, opts)
	}
}
