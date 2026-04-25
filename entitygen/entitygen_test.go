package entitygen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nurularifin27/schemagen/dbtype"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsManagedFile(t *testing.T) {
	managed := `// [SECTION: TABLE_NAME: START] - DO NOT REMOVE
// [SECTION: BASE: START] - DO NOT REMOVE`
	if !isManagedFile(managed) {
		t.Fatal("expected content to be treated as managed")
	}
	if isManagedFile("package entity\n") {
		t.Fatal("expected content without markers to be unmanaged")
	}
}

func TestHandleUnmanagedConflictSkip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.go")
	original := "package entity\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := handleUnmanagedConflict(path, "package entity\n// new\n", "users", "skip", Logger{}); err != nil {
		t.Fatalf("expected skip to succeed, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("expected original file to remain, got %q", string(got))
	}
}

func TestHandleUnmanagedConflictError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.go")
	if err := os.WriteFile(path, []byte("package entity\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := handleUnmanagedConflict(path, "package entity\n// new\n", "users", "error", Logger{})
	if err == nil || !strings.Contains(err.Error(), "unmanaged file conflict") {
		t.Fatalf("expected unmanaged conflict error, got %v", err)
	}
}

func TestHandleUnmanagedConflictBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.go")
	original := "package entity\n// old\n"
	rendered := "package entity\n// new\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := handleUnmanagedConflict(path, rendered, "users", "backup", Logger{}); err != nil {
		t.Fatalf("expected backup to succeed, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "// new") {
		t.Fatalf("expected rendered content, got %q", string(got))
	}

	matches, err := filepath.Glob(path + ".bak.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one backup file, got %d", len(matches))
	}
	backupContent, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(backupContent) != original {
		t.Fatalf("expected backup to keep original content, got %q", string(backupContent))
	}
}

func TestHandleUnmanagedConflictOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.go")
	rendered := "package entity\n// new\n"
	if err := os.WriteFile(path, []byte("package entity\n// old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := handleUnmanagedConflict(path, rendered, "users", "overwrite", Logger{}); err != nil {
		t.Fatalf("expected overwrite to succeed, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "// new") {
		t.Fatalf("expected rendered content, got %q", string(got))
	}
}

func TestBuildFieldsUsesMapper(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null, created_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}
	columnTypes, err := db.Migrator().ColumnTypes("users")
	if err != nil {
		t.Fatal(err)
	}

	fields, imports := buildFields("users", columnTypes, dbtype.New("sqlite"))
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}
	if fields[0].ColumnName != "id" {
		t.Fatalf("expected first field to carry column name, got %#v", fields[0])
	}
	if len(imports) == 0 {
		t.Fatal("expected generated imports for datetime field")
	}
}

func TestFieldNameFromColumnDoesNotSingularize(t *testing.T) {
	tests := map[string]string{
		"metadata":     "Metadata",
		"created_at":   "CreatedAt",
		"user_id":      "UserID",
		"product_sku":  "ProductSKU",
		"callback_url": "CallbackURL",
		"device_uuid":  "DeviceUUID",
		"raw_json":     "RawJSON",
	}

	for input, want := range tests {
		got := fieldNameFromColumn(input)
		if got != want {
			t.Fatalf("fieldNameFromColumn(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateCreatesManagedFile(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null, created_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	_, err = Generate(db, Options{
		Driver:     "sqlite",
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
	})
	if err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	path := filepath.Join(outDir, "user.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	required := []string{
		"Code generated by schemagen.",
		"Manual code is allowed outside managed SECTION markers.",
		"const TableNameUser = \"users\"",
		"type User struct {",
		"`db:\"created_at\" json:\"created_at\"`",
		"CreatedAt *time.Time",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("expected generated file to contain %q, got:\n%s", token, text)
		}
	}
}

func TestGenerateUsesMetadataFieldName(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, metadata text not null)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	_, err = Generate(db, Options{
		Driver:     "sqlite",
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
	})
	if err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "user.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "Metadata string") {
		t.Fatalf("expected Metadata field, got:\n%s", text)
	}
	if !strings.Contains(text, "`db:\"metadata\" json:\"metadata\"`") {
		t.Fatalf("expected sqlx tags, got:\n%s", text)
	}
	if strings.Contains(text, "Metadatum") {
		t.Fatalf("expected Metadatum to be absent, got:\n%s", text)
	}
}

func TestGeneratePreservesManualImportsOnManagedFile(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, created_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	path := filepath.Join(outDir, "user.go")
	managed := `package entity

import (
	"strings"
)

// [SECTION: TABLE_NAME: START] - DO NOT REMOVE
const TableNameUser = "users"

func (*User) TableName() string {
	return TableNameUser
}

// [SECTION: TABLE_NAME: END] - DO NOT REMOVE

type User struct {
	// [SECTION: BASE: START] - DO NOT REMOVE
	ID int64 ` + "`gorm:\"column:id\" json:\"id\"`" + `
	// [SECTION: BASE: END] - DO NOT REMOVE
}

func (u User) Slug() string {
	return strings.ToLower("X")
}
`
	if err := os.WriteFile(path, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = Generate(db, Options{
		Driver:     "sqlite",
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
	})
	if err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Count(text, "\"strings\"") != 1 {
		t.Fatalf("expected strings import exactly once, got:\n%s", text)
	}
	if strings.Count(text, "\"time\"") != 1 {
		t.Fatalf("expected time import exactly once, got:\n%s", text)
	}
	if !strings.Contains(text, "`db:\"created_at\" json:\"created_at\"`") {
		t.Fatalf("expected sqlx tag after regeneration, got:\n%s", text)
	}
	if !strings.Contains(text, "func (u User) Slug() string") {
		t.Fatalf("expected manual method to remain, got:\n%s", text)
	}
}

func TestGenerateAppliesTypeOverrides(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, amount decimal not null, payload json)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	_, err = Generate(db, Options{
		Driver:     "sqlite",
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
		TypeOverrides: []dbtype.Override{
			{DBType: "decimal", GoType: "money.Amount", Imports: []string{"example.com/money"}},
			{Table: "users", Column: "payload", GoType: "json.RawMessage", Imports: []string{"encoding/json"}},
		},
	})
	if err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "user.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	required := []string{
		`"encoding/json"`,
		`"example.com/money"`,
		"money.Amount",
		"*json.RawMessage",
		"`db:\"amount\" json:\"amount\"`",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("expected generated file to contain %q, got:\n%s", token, text)
		}
	}
}

func TestGenerateMatchesGoldenDefault(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null, created_at datetime, metadata json)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "user.go"), "testdata/user_default.golden")
}

func TestGenerateMatchesGoldenConfiguredStrategies(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE invoices (id integer primary key autoincrement, total decimal not null, payload json, issued_at datetime not null)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:          "sqlite",
		OutDir:          outDir,
		Tables:          []string{"invoices"},
		OnConflict:      "skip",
		DecimalStrategy: "string",
		JSONStrategy:    "rawmessage",
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "invoice.go"), "testdata/invoice_strategies.golden")
}

func TestGenerateMatchesGoldenOverrides(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE orders (id integer primary key autoincrement, amount decimal not null, payload json, external_id uuid)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		OutDir:     outDir,
		Tables:     []string{"orders"},
		OnConflict: "skip",
		TypeOverrides: []dbtype.Override{
			{DBType: "decimal", GoType: "money.Amount", Imports: []string{"example.com/project/money"}},
			{Table: "orders", Column: "payload", GoType: "json.RawMessage", Imports: []string{"encoding/json"}},
			{Column: "external_id", GoType: "uuid.UUID", Imports: []string{"github.com/google/uuid"}},
		},
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "order.go"), "testdata/order_overrides.golden")
}

func TestGenerateMatchesGoldenGORMRenderer(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE accounts (id integer primary key autoincrement, email text not null, created_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		Renderer:   RendererGORM,
		OutDir:     outDir,
		Tables:     []string{"accounts"},
		OnConflict: "skip",
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "account.go"), "testdata/account_gorm.golden")
}

func TestGenerateMatchesGoldenSQLNullStrategy(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE events (id integer primary key autoincrement, name text, occurred_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:           "sqlite",
		Renderer:         RendererSQLX,
		OutDir:           outDir,
		Tables:           []string{"events"},
		OnConflict:       "skip",
		NullableStrategy: "sqlnull",
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "event.go"), "testdata/event_sqlnull.golden")
}

func TestGenerateMatchesGoldenRelationsSQLX(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE orders (id integer primary key autoincrement, user_id integer not null)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		Renderer:   RendererSQLX,
		OutDir:     outDir,
		Tables:     []string{"orders"},
		OnConflict: "skip",
		Relations: []Relation{{
			Table:       "orders",
			Kind:        "belongs_to",
			Field:       "User",
			TargetTable: "users",
			ForeignKey:  "user_id",
			TargetKey:   "id",
		}},
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "order.go"), "testdata/order_relation_sqlx.golden")
}

func TestGenerateMatchesGoldenRelationsGORM(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE orders (id integer primary key autoincrement, user_id integer not null)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		Renderer:   RendererGORM,
		OutDir:     outDir,
		Tables:     []string{"orders"},
		OnConflict: "skip",
		Relations: []Relation{{
			Table:       "orders",
			Kind:        "belongs_to",
			Field:       "User",
			TargetTable: "users",
			ForeignKey:  "user_id",
			TargetKey:   "id",
		}},
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "order.go"), "testdata/order_relation_gorm.golden")
}

func TestGenerateMatchesGoldenManyToManyPivotSQLX(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE roles (id integer primary key autoincrement, name text not null)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_roles (id integer primary key autoincrement, user_id integer not null, role_id integer not null, assigned_at datetime null)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		Renderer:   RendererSQLX,
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
		Relations: []Relation{{
			Table:          "users",
			Kind:           "many_to_many",
			Field:          "Roles",
			PivotField:     "UserRoles",
			TargetTable:    "roles",
			JoinTable:      "user_roles",
			JoinForeignKey: "user_id",
			JoinTargetKey:  "role_id",
			SourceKey:      "id",
			TargetKey:      "id",
		}},
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "user.go"), "testdata/user_many_to_many_pivot_sqlx.golden")
}

func TestGenerateMatchesGoldenManyToManyPivotGORM(t *testing.T) {
	db, err := openSQLiteDB(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key autoincrement, name text not null)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE roles (id integer primary key autoincrement, name text not null)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_roles (id integer primary key autoincrement, user_id integer not null, role_id integer not null, assigned_at datetime null)`).Error; err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if _, err := Generate(db, Options{
		Driver:     "sqlite",
		Renderer:   RendererGORM,
		OutDir:     outDir,
		Tables:     []string{"users"},
		OnConflict: "skip",
		Relations: []Relation{{
			Table:          "users",
			Kind:           "many_to_many",
			Field:          "Roles",
			PivotField:     "UserRoles",
			TargetTable:    "roles",
			JoinTable:      "user_roles",
			JoinForeignKey: "user_id",
			JoinTargetKey:  "role_id",
			SourceKey:      "id",
			TargetKey:      "id",
		}},
	}); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}

	assertMatchesGolden(t, filepath.Join(outDir, "user.go"), "testdata/user_many_to_many_pivot_gorm.golden")
}

func assertMatchesGolden(t *testing.T, gotPath, goldenPath string) {
	t.Helper()

	got, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("generated output mismatch for %s\nwant:\n%s\ngot:\n%s", gotPath, string(want), string(got))
	}
}

func openSQLiteDB(t *testing.T) (*gorm.DB, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), fmt.Sprintf("%s.db", strings.ReplaceAll(t.Name(), "/", "_")))
	return gorm.Open(sqlite.Open(path), &gorm.Config{})
}
