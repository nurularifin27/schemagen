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
	switch dbType := strings.ToLower(column.DatabaseType); dbType {
	case "tinyint":
		return mysqlTinyIntType(fullType), nil
	case "smallint":
		return mysqlSignedOrUnsigned(fullType, "int16", "uint16"), nil
	case "mediumint", "int", "integer":
		return mysqlSignedOrUnsigned(fullType, "int32", "uint32"), nil
	case "bigint":
		return mysqlSignedOrUnsigned(fullType, "int64", "uint64"), nil
	case "decimal", "numeric":
		return decimalType(opts), nil
	case "json":
		return jsonType(opts)
	default:
		if match, ok := mysqlSimpleTypes[dbType]; ok {
			return match.goType, match.imports
		}
		return commonTypeFallback(column, opts)
	}
}

func mysqlTinyIntType(fullType string) string {
	if strings.Contains(fullType, "(1)") {
		return "bool"
	}
	return mysqlSignedOrUnsigned(fullType, "int8", "uint8")
}

func mysqlSignedOrUnsigned(fullType, signedType, unsignedType string) string {
	if hasUnsigned(fullType) {
		return unsignedType
	}
	return signedType
}

var mysqlSimpleTypes = map[string]scanTypeMatch{
	"float":              {goType: "float32"},
	"double":             {goType: "float64"},
	"double precision":   {goType: "float64"},
	"real":               {goType: "float64"},
	"bit":                {goType: "[]byte"},
	"bool":               {goType: "bool"},
	"boolean":            {goType: "bool"},
	"char":               {goType: "string"},
	"varchar":            {goType: "string"},
	"tinytext":           {goType: "string"},
	"text":               {goType: "string"},
	"mediumtext":         {goType: "string"},
	"longtext":           {goType: "string"},
	"enum":               {goType: "string"},
	"set":                {goType: "string"},
	"date":               {goType: "time.Time", imports: []string{`"time"`}},
	"datetime":           {goType: "time.Time", imports: []string{`"time"`}},
	"timestamp":          {goType: "time.Time", imports: []string{`"time"`}},
	"time":               {goType: "string"},
	"year":               {goType: "int16"},
	"binary":             {goType: "[]byte"},
	"varbinary":          {goType: "[]byte"},
	"tinyblob":           {goType: "[]byte"},
	"blob":               {goType: "[]byte"},
	"mediumblob":         {goType: "[]byte"},
	"longblob":           {goType: "[]byte"},
	"geometry":           {goType: "[]byte"},
	"point":              {goType: "[]byte"},
	"linestring":         {goType: "[]byte"},
	"polygon":            {goType: "[]byte"},
	"multipoint":         {goType: "[]byte"},
	"multilinestring":    {goType: "[]byte"},
	"multipolygon":       {goType: "[]byte"},
	"geometrycollection": {goType: "[]byte"},
}
