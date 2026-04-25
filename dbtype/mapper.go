package dbtype

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type Column struct {
	TableName     string
	Name          string
	DatabaseType  string
	FullType      string
	Nullable      bool
	HasNullable   bool
	PrimaryKey    bool
	HasPrimaryKey bool
	AutoIncrement bool
	HasAutoInc    bool
	Unique        bool
	HasUnique     bool
	Length        int64
	HasLength     bool
	Precision     int64
	Scale         int64
	HasDecimal    bool
	ScanType      reflect.Type
	DefaultValue  string
	HasDefault    bool
}

type Field struct {
	Name    string
	GoType  string
	Tags    []string
	Imports []string
}

type Options struct {
	DecimalStrategy  string
	JSONStrategy     string
	NullableStrategy string
	Overrides        []Override
}

type Override struct {
	Table   string
	Column  string
	DBType  string
	GoType  string
	Imports []string
}

type Mapper interface {
	Map(column Column, fieldName string) Field
}

type mapper struct {
	driver driverMapper
	opts   Options
}

type driverMapper interface {
	goType(column Column, opts Options) (string, []string)
}

const (
	DecimalStrategyFloat64  = "float64"
	DecimalStrategyString   = "string"
	JSONStrategyBytes       = "bytes"
	JSONStrategyRawMessage  = "rawmessage"
	NullableStrategyPointer = "pointer"
	NullableStrategySQLNull = "sqlnull"
)

func New(driver string, rawOpts ...any) Mapper {
	opts := Options{
		DecimalStrategy:  DecimalStrategyFloat64,
		JSONStrategy:     JSONStrategyBytes,
		NullableStrategy: NullableStrategyPointer,
	}
	if len(rawOpts) > 0 {
		if decimal, ok := rawOpts[0].(string); ok && strings.TrimSpace(decimal) != "" {
			opts.DecimalStrategy = strings.ToLower(strings.TrimSpace(decimal))
		}
	}
	if len(rawOpts) > 1 {
		if json, ok := rawOpts[1].(string); ok && strings.TrimSpace(json) != "" {
			opts.JSONStrategy = strings.ToLower(strings.TrimSpace(json))
		}
	}
	if len(rawOpts) > 2 {
		if nullable, ok := rawOpts[2].(string); ok && strings.TrimSpace(nullable) != "" {
			opts.NullableStrategy = strings.ToLower(strings.TrimSpace(nullable))
		}
	}
	if len(rawOpts) > 3 {
		if overrides, ok := rawOpts[3].([]Override); ok {
			opts.Overrides = overrides
		}
	}

	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres":
		return mapper{driver: postgresMapper{}, opts: opts}
	case "mysql":
		return mapper{driver: mysqlMapper{}, opts: opts}
	case "mariadb":
		return mapper{driver: mariaDBMapper{}, opts: opts}
	case "sqlite", "sqlite3":
		return mapper{driver: sqliteMapper{}, opts: opts}
	default:
		return mapper{driver: defaultMapper{}, opts: opts}
	}
}

func FromGormColumn(col gorm.ColumnType) Column {
	result := Column{
		Name:         col.Name(),
		DatabaseType: strings.ToLower(strings.TrimSpace(col.DatabaseTypeName())),
		ScanType:     col.ScanType(),
	}
	if fullType, ok := col.ColumnType(); ok {
		result.FullType = strings.ToLower(strings.TrimSpace(fullType))
	}
	if nullable, ok := col.Nullable(); ok {
		result.Nullable = nullable
		result.HasNullable = true
	}
	if primaryKey, ok := col.PrimaryKey(); ok {
		result.PrimaryKey = primaryKey
		result.HasPrimaryKey = true
	}
	if autoIncrement, ok := col.AutoIncrement(); ok {
		result.AutoIncrement = autoIncrement
		result.HasAutoInc = true
	}
	if unique, ok := col.Unique(); ok {
		result.Unique = unique
		result.HasUnique = true
	}
	if length, ok := col.Length(); ok {
		result.Length = length
		result.HasLength = true
	}
	if precision, scale, ok := col.DecimalSize(); ok {
		result.Precision = precision
		result.Scale = scale
		result.HasDecimal = true
	}
	if defaultValue, ok := col.DefaultValue(); ok {
		result.DefaultValue = defaultValue
		result.HasDefault = true
	}
	return result
}

func (m mapper) Map(column Column, fieldName string) Field {
	if override, ok := matchOverride(column, m.opts.Overrides); ok {
		goType := override.GoType
		goType, imports := nullableGoType(column, goType, override.Imports, m.opts)
		return buildField(column, fieldName, goType, imports)
	}

	goType, imports := m.driver.goType(column, m.opts)
	goType, imports = nullableGoType(column, goType, imports, m.opts)

	return buildField(column, fieldName, goType, imports)
}

func nullableGoType(column Column, goType string, imports []string, opts Options) (string, []string) {
	if sqlNullType, sqlNullImports, ok := sqlNullTypeFor(column, goType, opts); ok {
		return sqlNullType, sqlNullImports
	}
	if shouldUsePointer(column, goType) {
		goType = "*" + goType
	}
	return goType, imports
}

func buildField(column Column, fieldName, goType string, imports []string) Field {
	normalizedImports := normalizeImports(imports)

	tags := []string{
		fmt.Sprintf("column:%s", column.Name),
	}
	if column.PrimaryKey {
		tags = append(tags, "primaryKey")
	}
	if column.AutoIncrement {
		tags = append(tags, "autoIncrement")
	}
	if column.HasNullable && !column.Nullable {
		tags = append(tags, "not null")
	}
	if column.Unique {
		tags = append(tags, "uniqueIndex")
	}
	if column.HasDefault {
		tags = append(tags, fmt.Sprintf("default:%s", sanitizeDefault(column.DefaultValue)))
	}

	return Field{
		Name:    fieldName,
		GoType:  goType,
		Tags:    tags,
		Imports: normalizedImports,
	}
}

func typeFromScanType(t reflect.Type) (string, []string, bool) {
	if t == nil {
		return "", nil, false
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return "[]byte", nil, true
	}

	switch t.Kind() {
	case reflect.Bool:
		return "bool", nil, true
	case reflect.Int:
		return "int", nil, true
	case reflect.Int8:
		return "int8", nil, true
	case reflect.Int16:
		return "int16", nil, true
	case reflect.Int32:
		return "int32", nil, true
	case reflect.Int64:
		return "int64", nil, true
	case reflect.Uint:
		return "uint", nil, true
	case reflect.Uint8:
		return "uint8", nil, true
	case reflect.Uint16:
		return "uint16", nil, true
	case reflect.Uint32:
		return "uint32", nil, true
	case reflect.Uint64:
		return "uint64", nil, true
	case reflect.Float32:
		return "float32", nil, true
	case reflect.Float64:
		return "float64", nil, true
	case reflect.String:
		return "string", nil, true
	}

	pkg := t.PkgPath()
	name := t.Name()
	full := pkg + "." + name
	switch full {
	case "time.Time":
		return "time.Time", []string{`"time"`}, true
	case "database/sql.NullString":
		return "string", nil, true
	case "database/sql.NullBool":
		return "bool", nil, true
	case "database/sql.NullByte":
		return "int8", nil, true
	case "database/sql.NullInt16":
		return "int16", nil, true
	case "database/sql.NullInt32":
		return "int32", nil, true
	case "database/sql.NullInt64":
		return "int64", nil, true
	case "database/sql.NullFloat64":
		return "float64", nil, true
	case "database/sql.NullTime":
		return "time.Time", []string{`"time"`}, true
	}

	if name == "UUID" && strings.Contains(pkg, "uuid") {
		return "string", nil, true
	}
	if strings.Contains(strings.ToLower(name), "time") && strings.Contains(pkg, "datatypes") {
		return "time.Time", []string{`"time"`}, true
	}
	if strings.Contains(strings.ToLower(name), "json") && strings.Contains(pkg, "datatypes") {
		return "[]byte", nil, true
	}

	return "", nil, false
}

func shouldUsePointer(column Column, goType string) bool {
	if strings.ToLower(strings.TrimSpace(goType)) == "interface{}" {
		return false
	}
	if !column.Nullable {
		return false
	}
	if strings.HasPrefix(goType, "*") {
		return false
	}
	if strings.HasPrefix(goType, "[]") {
		return false
	}
	return true
}

func sanitizeDefault(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ";", "")
	return value
}

func unsignedType(bits int) string {
	switch bits {
	case 8:
		return "uint8"
	case 16:
		return "uint16"
	case 32:
		return "uint32"
	case 64:
		return "uint64"
	default:
		return "uint"
	}
}

func hasUnsigned(fullType string) bool {
	return strings.Contains(strings.ToLower(fullType), "unsigned")
}

func inferByScanType(column Column) (string, []string, bool) {
	return typeFromScanType(column.ScanType)
}

func decimalType(opts Options) string {
	if opts.DecimalStrategy == DecimalStrategyString {
		return "string"
	}
	return "float64"
}

func jsonType(opts Options) (string, []string) {
	if opts.JSONStrategy == JSONStrategyRawMessage {
		return "json.RawMessage", []string{`"encoding/json"`}
	}
	return "[]byte", nil
}

func sqlNullTypeFor(column Column, goType string, opts Options) (string, []string, bool) {
	if opts.NullableStrategy != NullableStrategySQLNull || !column.Nullable {
		return "", nil, false
	}
	if column.PrimaryKey || column.AutoIncrement {
		return "", nil, false
	}
	switch goType {
	case "string":
		return "sql.NullString", []string{`"database/sql"`}, true
	case "bool":
		return "sql.NullBool", []string{`"database/sql"`}, true
	case "int16":
		return "sql.NullInt16", []string{`"database/sql"`}, true
	case "int32":
		return "sql.NullInt32", []string{`"database/sql"`}, true
	case "int64":
		return "sql.NullInt64", []string{`"database/sql"`}, true
	case "float64":
		return "sql.NullFloat64", []string{`"database/sql"`}, true
	case "time.Time":
		return "sql.NullTime", []string{`"database/sql"`}, true
	default:
		return "", nil, false
	}
}

func matchOverride(column Column, overrides []Override) (Override, bool) {
	bestScore := -1
	var best Override
	for _, override := range overrides {
		score, ok := overrideMatchScore(column, override)
		if !ok {
			continue
		}
		if score > bestScore {
			bestScore = score
			best = override
		}
	}
	return best, bestScore >= 0
}

func overrideMatchScore(column Column, override Override) (int, bool) {
	if strings.TrimSpace(override.GoType) == "" {
		return 0, false
	}

	score := 0
	if override.Table != "" {
		if !strings.EqualFold(strings.TrimSpace(override.Table), column.TableName) {
			return 0, false
		}
		score += 4
	}
	if override.Column != "" {
		if !strings.EqualFold(strings.TrimSpace(override.Column), column.Name) {
			return 0, false
		}
		score += 2
	}
	if override.DBType != "" {
		if !strings.EqualFold(strings.TrimSpace(override.DBType), column.DatabaseType) {
			return 0, false
		}
		score++
	}
	return score, score > 0
}

func normalizeImports(imports []string) []string {
	if len(imports) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(imports))
	normalized := make([]string, 0, len(imports))
	for _, imp := range imports {
		imp = strings.TrimSpace(imp)
		if imp == "" {
			continue
		}
		if !strings.HasPrefix(imp, `"`) {
			imp = `"` + imp + `"`
		}
		if seen[imp] {
			continue
		}
		seen[imp] = true
		normalized = append(normalized, imp)
	}
	return normalized
}
