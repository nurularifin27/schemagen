package dbtype

import (
	"fmt"
	"reflect"
	"regexp"
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

type Mapper struct {
	driver string
}

func New(driver string) Mapper {
	return Mapper{driver: strings.ToLower(driver)}
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
	if column.FullType != "" {
		tags = append(tags, fmt.Sprintf("type:%s", column.FullType))
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
	if arrayType, imports, ok := m.arrayType(column); ok {
		return arrayType, imports
	}

	switch canonicalType(column) {
	case "bool":
		return "bool", nil
	case "smallint":
		return "int16", nil
	case "integer":
		return "int32", nil
	case "bigint":
		return "int64", nil
	case "float":
		if column.DatabaseType == "float4" {
			return "float32", nil
		}
		return "float64", nil
	case "decimal":
		return "decimal.Decimal", []string{`"github.com/shopspring/decimal"`}
	case "bytes":
		return "[]byte", nil
	case "json":
		return "datatypes.JSON", []string{`"gorm.io/datatypes"`}
	case "uuid":
		return "uuid.UUID", []string{`"github.com/google/uuid"`}
	case "date":
		return "datatypes.Date", []string{`"gorm.io/datatypes"`}
	case "time":
		return "datatypes.Time", []string{`"gorm.io/datatypes"`}
	case "datetime":
		return "time.Time", []string{`"time"`}
	default:
		return "string", nil
	}
}

func (m Mapper) arrayType(column Column) (string, []string, bool) {
	full := column.FullType
	if !strings.Contains(full, "[]") && !strings.HasPrefix(column.DatabaseType, "_") {
		return "", nil, false
	}

	element := postgresArrayElement(column)
	switch element {
	case "uuid":
		return "pgtype.FlatArray[uuid.UUID]", []string{
			`"github.com/google/uuid"`,
			`"github.com/jackc/pgx/v5/pgtype"`,
		}, true
	case "bool":
		return "pgtype.FlatArray[bool]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	case "int2":
		return "pgtype.FlatArray[int16]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	case "int4":
		return "pgtype.FlatArray[int32]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	case "int8":
		return "pgtype.FlatArray[int64]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	case "float4":
		return "pgtype.FlatArray[float32]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	case "float8", "numeric":
		return "pgtype.FlatArray[float64]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	default:
		return "pgtype.FlatArray[string]", []string{`"github.com/jackc/pgx/v5/pgtype"`}, true
	}
}

func (m Mapper) shouldUsePointer(column Column, goType string) bool {
	if !column.Nullable {
		return false
	}

	if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "pgtype.FlatArray[") {
		return false
	}
	if goType == "datatypes.JSON" {
		return false
	}
	return true
}

func canonicalType(column Column) string {
	dbType := strings.ToLower(column.DatabaseType)
	full := strings.ToLower(column.FullType)

	switch {
	case dbType == "bool", dbType == "boolean":
		return "bool"
	case dbType == "tinyint" && strings.Contains(full, "(1)"):
		return "bool"
	case dbType == "bit" && strings.Contains(full, "(1)"):
		return "bool"
	case dbType == "smallint", dbType == "int2":
		return "smallint"
	case dbType == "integer", dbType == "int", dbType == "serial", dbType == "int4", dbType == "mediumint":
		return "integer"
	case dbType == "bigint", dbType == "bigserial", dbType == "int8":
		return "bigint"
	case dbType == "real", dbType == "double", dbType == "double precision", dbType == "float", dbType == "float4", dbType == "float8":
		return "float"
	case dbType == "numeric", dbType == "decimal", strings.HasPrefix(full, "numeric"), strings.HasPrefix(full, "decimal"):
		return "decimal"
	case dbType == "json", dbType == "jsonb":
		return "json"
	case dbType == "uuid":
		return "uuid"
	case dbType == "date":
		return "date"
	case dbType == "time", dbType == "timetz", strings.HasPrefix(full, "time without time zone"), strings.HasPrefix(full, "time with time zone"):
		return "time"
	case dbType == "timestamp", dbType == "timestamptz", dbType == "datetime", strings.HasPrefix(full, "timestamp"), strings.HasPrefix(full, "datetime"):
		return "datetime"
	case dbType == "bytea", dbType == "blob", dbType == "binary", dbType == "varbinary":
		return "bytes"
	default:
		return "string"
	}
}

func postgresArrayElement(column Column) string {
	dbType := strings.TrimPrefix(strings.ToLower(column.DatabaseType), "_")
	if dbType != column.DatabaseType {
		return dbType
	}

	full := strings.ToLower(column.FullType)
	re := regexp.MustCompile(`^([a-z0-9_ ]+)\[\]`)
	if matches := re.FindStringSubmatch(full); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return dbType
}

func sanitizeDefault(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ";", "")
	return value
}
