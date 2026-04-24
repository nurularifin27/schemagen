package dbtype

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type Column struct {
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

type Mapper struct{}

func New(_ string, _ ...string) Mapper {
	return Mapper{}
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

func (m Mapper) Map(column Column, fieldName string) Field {
	goType, imports := m.goType(column)
	if m.shouldUsePointer(column, goType) {
		goType = "*" + goType
	}

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
		Imports: imports,
	}
}

func (m Mapper) goType(column Column) (string, []string) {
	if scanType := column.ScanType; scanType != nil {
		if scanType.Kind() == reflect.Pointer {
			scanType = scanType.Elem()
		}
		if goType, imports, ok := typeFromScanType(scanType); ok {
			return goType, imports
		}
	}

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
	case "integer", "int", "serial", "int4", "mediumint":
		return "int32", nil
	case "bigint", "bigserial", "int8":
		return "int64", nil
	case "real", "double", "double precision", "float", "float4", "float8", "numeric", "decimal":
		return "float64", nil
	case "bytea", "blob", "binary", "varbinary":
		return "[]byte", nil
	case "date", "datetime", "timestamp", "timestamptz":
		return "time.Time", []string{`"time"`}
	default:
		return "string", nil
	}
}

func typeFromScanType(t reflect.Type) (string, []string, bool) {
	if t == nil {
		return "", nil, false
	}

	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return "[]byte", nil, true
	}

	switch t.Kind() {
	case reflect.Bool:
		return "bool", nil, true
	case reflect.Int, reflect.Int8:
		return "int8", nil, true
	case reflect.Int16:
		return "int16", nil, true
	case reflect.Int32:
		return "int32", nil, true
	case reflect.Int64:
		return "int64", nil, true
	case reflect.Uint, reflect.Uint8:
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

func (m Mapper) shouldUsePointer(column Column, goType string) bool {
	if !column.Nullable {
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
