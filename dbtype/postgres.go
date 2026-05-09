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

	switch dbType := strings.ToLower(column.DatabaseType); dbType {
	case "numeric", "decimal":
		return decimalType(opts), nil
	case "json", "jsonb":
		return jsonType(opts)
	default:
		if strings.HasPrefix(dbType, "_") {
			return postgresArrayType(dbType), nil
		}
		if match, ok := postgresSimpleTypes[dbType]; ok {
			return match.goType, match.imports
		}
		return commonTypeFallback(column, opts)
	}
}

var postgresSimpleTypes = map[string]scanTypeMatch{
	"smallint":                    {goType: "int16"},
	"int2":                        {goType: "int16"},
	"integer":                     {goType: "int32"},
	"int":                         {goType: "int32"},
	"int4":                        {goType: "int32"},
	"serial":                      {goType: "int32"},
	"serial4":                     {goType: "int32"},
	"bigint":                      {goType: "int64"},
	"int8":                        {goType: "int64"},
	"bigserial":                   {goType: "int64"},
	"serial8":                     {goType: "int64"},
	"real":                        {goType: "float32"},
	"float4":                      {goType: "float32"},
	"double precision":            {goType: "float64"},
	"float8":                      {goType: "float64"},
	"boolean":                     {goType: "bool"},
	"bool":                        {goType: "bool"},
	"char":                        {goType: "string"},
	"character":                   {goType: "string"},
	"bpchar":                      {goType: "string"},
	"varchar":                     {goType: "string"},
	"character varying":           {goType: "string"},
	"text":                        {goType: "string"},
	"citext":                      {goType: "string"},
	"name":                        {goType: "string"},
	"date":                        {goType: "time.Time", imports: []string{`"time"`}},
	"time":                        {goType: "time.Time", imports: []string{`"time"`}},
	"time without time zone":      {goType: "time.Time", imports: []string{`"time"`}},
	"timetz":                      {goType: "time.Time", imports: []string{`"time"`}},
	"time with time zone":         {goType: "time.Time", imports: []string{`"time"`}},
	"timestamp":                   {goType: "time.Time", imports: []string{`"time"`}},
	"timestamp without time zone": {goType: "time.Time", imports: []string{`"time"`}},
	"timestamptz":                 {goType: "time.Time", imports: []string{`"time"`}},
	"timestamp with time zone":    {goType: "time.Time", imports: []string{`"time"`}},
	"interval":                    {goType: "string"},
	"bytea":                       {goType: "[]byte"},
	"uuid":                        {goType: "string"},
	"inet":                        {goType: "string"},
	"cidr":                        {goType: "string"},
	"macaddr":                     {goType: "string"},
	"macaddr8":                    {goType: "string"},
	"xml":                         {goType: "string"},
	"money":                       {goType: "string"},
	"tsvector":                    {goType: "string"},
	"tsquery":                     {goType: "string"},
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
