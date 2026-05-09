package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfigIfExistsMissingFile(t *testing.T) {
	cfg, err := loadConfigIfExists(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg, Config{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadConfigIfExistsReadsYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.yaml")
	if err := os.WriteFile(path, []byte(sampleConfigYAML()), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfigIfExists(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSampleConfig(t, cfg)
}

func TestNormalizeConfigAppliesDefaults(t *testing.T) {
	cfg := Config{}
	normalizeConfig(&cfg)

	if cfg.Driver != defaultDriver {
		t.Fatalf("expected driver %q, got %q", defaultDriver, cfg.Driver)
	}
	if cfg.OutDir != defaultOutDir {
		t.Fatalf("expected out dir %q, got %q", defaultOutDir, cfg.OutDir)
	}
	if cfg.Renderer != defaultRenderer {
		t.Fatalf("expected renderer %q, got %q", defaultRenderer, cfg.Renderer)
	}
	if cfg.OnConflict != defaultOnConflict {
		t.Fatalf("expected on conflict %q, got %q", defaultOnConflict, cfg.OnConflict)
	}
	if cfg.DecimalStrategy != defaultDecimalStrategy {
		t.Fatalf("expected decimal strategy %q, got %q", defaultDecimalStrategy, cfg.DecimalStrategy)
	}
	if cfg.JSONStrategy != defaultJSONStrategy {
		t.Fatalf("expected json strategy %q, got %q", defaultJSONStrategy, cfg.JSONStrategy)
	}
	if cfg.JSONCaseStrategy != defaultJSONCaseStrategy {
		t.Fatalf("expected json case strategy %q, got %q", defaultJSONCaseStrategy, cfg.JSONCaseStrategy)
	}
	if cfg.NullableStrategy != defaultNullableStrategy {
		t.Fatalf("expected nullable strategy %q, got %q", defaultNullableStrategy, cfg.NullableStrategy)
	}
	if !reflect.DeepEqual(cfg.Exclude, defaultExclude) {
		t.Fatalf("expected default exclude %#v, got %#v", defaultExclude, cfg.Exclude)
	}
}

func TestNormalizeConfigNormalizesCase(t *testing.T) {
	cfg := Config{
		Driver:           "Postgres",
		Renderer:         "GORM",
		OnConflict:       "BACKUP",
		DecimalStrategy:  "String",
		JSONStrategy:     "RAWMESSAGE",
		JSONCaseStrategy: "Camel",
		NullableStrategy: "SQLNULL",
	}
	normalizeConfig(&cfg)

	if cfg.Driver != "postgres" {
		t.Fatalf("expected normalized driver, got %q", cfg.Driver)
	}
	if cfg.Renderer != "gorm" {
		t.Fatalf("expected normalized renderer, got %q", cfg.Renderer)
	}
	if cfg.OnConflict != "backup" {
		t.Fatalf("expected normalized on conflict, got %q", cfg.OnConflict)
	}
	if cfg.DecimalStrategy != "string" {
		t.Fatalf("expected normalized decimal strategy, got %q", cfg.DecimalStrategy)
	}
	if cfg.JSONStrategy != "rawmessage" {
		t.Fatalf("expected normalized json strategy, got %q", cfg.JSONStrategy)
	}
	if cfg.JSONCaseStrategy != "camel" {
		t.Fatalf("expected normalized json case strategy, got %q", cfg.JSONCaseStrategy)
	}
	if cfg.NullableStrategy != "sqlnull" {
		t.Fatalf("expected normalized nullable strategy, got %q", cfg.NullableStrategy)
	}
}

func TestMergeConfigPrefersOverride(t *testing.T) {
	base := Config{
		DSN:              "dsn-base",
		Driver:           "postgres",
		Renderer:         "sqlx",
		OutDir:           "./base",
		Tables:           []string{"users"},
		Exclude:          []string{"migrations"},
		OnConflict:       "skip",
		DecimalStrategy:  "float64",
		JSONStrategy:     "bytes",
		JSONCaseStrategy: "snake",
		NullableStrategy: "pointer",
	}
	override := Config{
		DSN:              "dsn-override",
		Renderer:         "gorm",
		OutDir:           "./override",
		Tables:           []string{"companies"},
		OnConflict:       "backup",
		DecimalStrategy:  "string",
		JSONCaseStrategy: "snake",
		NullableStrategy: "sqlnull",
	}

	cfg := mergeConfig(base, override)
	if cfg.DSN != "dsn-override" || cfg.OutDir != "./override" || cfg.Driver != "postgres" || cfg.Renderer != "gorm" || cfg.OnConflict != "backup" || cfg.DecimalStrategy != "string" || cfg.JSONStrategy != "bytes" || cfg.JSONCaseStrategy != "snake" || cfg.NullableStrategy != "sqlnull" {
		t.Fatalf("unexpected merged config: %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.Tables, []string{"companies"}) {
		t.Fatalf("unexpected merged tables: %#v", cfg.Tables)
	}
	if !reflect.DeepEqual(cfg.Exclude, []string{"migrations"}) {
		t.Fatalf("unexpected merged exclude: %#v", cfg.Exclude)
	}
}

func TestIsValidConflictPolicy(t *testing.T) {
	valid := []string{"skip", "error", "backup", "overwrite"}
	for _, policy := range valid {
		if !isValidConflictPolicy(policy) {
			t.Fatalf("expected policy %q to be valid", policy)
		}
	}
	if isValidConflictPolicy("random") {
		t.Fatal("expected random policy to be invalid")
	}
}

func TestStrategyValidators(t *testing.T) {
	testValidator(t, "renderer", isValidRenderer, []string{"plain", "sqlx", "gorm"}, []string{"ent"})
	testValidator(t, "decimal", isValidDecimalStrategy, []string{"float64", "string"}, []string{"decimal"})
	testValidator(t, "nullable", isValidNullableStrategy, []string{"pointer", "sqlnull"}, []string{"ptr"})
	testValidator(t, "json", isValidJSONStrategy, []string{"bytes", "rawmessage"}, []string{"datatypes"})
	testValidator(t, "json-case", isValidJSONCaseStrategy, []string{"snake", "camel"}, []string{"mixed"})
}

func sampleConfigYAML() string {
	return "dsn: postgres://x\n" +
		"driver: postgres\n" +
		"renderer: gorm\n" +
		"out_dir: ./entity\n" +
		"tables:\n  - users\n" +
		"exclude:\n  - migrations\n" +
		"on_conflict: backup\n" +
		"decimal_strategy: string\n" +
		"json_strategy: rawmessage\n" +
		"json_case_strategy: camel\n" +
		"nullable_strategy: sqlnull\n" +
		"type_overrides:\n  - db_type: uuid\n    go_type: github.com/google/uuid.UUID\n    imports:\n      - github.com/google/uuid\n"
}

func assertSampleConfig(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.DSN != "postgres://x" || cfg.Driver != "postgres" || cfg.Renderer != "gorm" || cfg.OutDir != "./entity" {
		t.Fatalf("unexpected base config: %#v", cfg)
	}
	if cfg.OnConflict != "backup" || cfg.DecimalStrategy != "string" || cfg.JSONStrategy != "rawmessage" || cfg.JSONCaseStrategy != "camel" || cfg.NullableStrategy != "sqlnull" {
		t.Fatalf("unexpected strategy config: %#v", cfg)
	}
	if len(cfg.TypeOverrides) != 1 || cfg.TypeOverrides[0].GoType != "github.com/google/uuid.UUID" {
		t.Fatalf("unexpected type overrides: %#v", cfg.TypeOverrides)
	}
	if !reflect.DeepEqual(cfg.Tables, []string{"users"}) {
		t.Fatalf("unexpected tables: %#v", cfg.Tables)
	}
	if !reflect.DeepEqual(cfg.Exclude, []string{"migrations"}) {
		t.Fatalf("unexpected exclude: %#v", cfg.Exclude)
	}
}

func testValidator(t *testing.T, name string, fn func(string) bool, valid []string, invalid []string) {
	t.Helper()
	for _, value := range valid {
		if !fn(value) {
			t.Fatalf("expected %s %q to be valid", name, value)
		}
	}
	for _, value := range invalid {
		if fn(value) {
			t.Fatalf("expected %s %q to be invalid", name, value)
		}
	}
}

func TestNormalizeTypeOverrides(t *testing.T) {
	overrides := normalizeTypeOverrides([]TypeOverrideConfig{{
		Table:   " Users ",
		Column:  " Payload ",
		DBType:  " JSONB ",
		GoType:  " json.RawMessage ",
		Imports: []string{" encoding/json "},
	}})
	if len(overrides) != 1 {
		t.Fatalf("expected one override, got %d", len(overrides))
	}
	if overrides[0].Table != "users" || overrides[0].Column != "payload" || overrides[0].DBType != "jsonb" || overrides[0].GoType != "json.RawMessage" || overrides[0].Imports[0] != "encoding/json" {
		t.Fatalf("unexpected normalized override: %#v", overrides[0])
	}
}

func TestValidateTypeOverrides(t *testing.T) {
	if err := validateTypeOverrides([]TypeOverrideConfig{{GoType: "string", DBType: "uuid"}}); err != nil {
		t.Fatalf("expected override to be valid, got %v", err)
	}
	if err := validateTypeOverrides([]TypeOverrideConfig{{DBType: "uuid"}}); err == nil {
		t.Fatal("expected missing go_type to fail")
	}
	if err := validateTypeOverrides([]TypeOverrideConfig{{GoType: "string"}}); err == nil {
		t.Fatal("expected missing matchers to fail")
	}
}

func TestDefaultConfigTemplateIncludesDefaults(t *testing.T) {
	content := defaultConfigTemplate()

	required := []string{
		"dsn: \"\"",
		"driver: postgres",
		"renderer: sqlx",
		"out_dir: ./internal/entity",
		"on_conflict: skip",
		"decimal_strategy: float64",
		"json_strategy: bytes",
		"json_case_strategy: snake",
		"nullable_strategy: pointer",
		"type_overrides: []",
		"schema_migrations",
		"goose_db_version",
		"migrations",
	}
	for _, token := range required {
		if !strings.Contains(content, token) {
			t.Fatalf("expected config template to contain %q, got:\n%s", token, content)
		}
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schemagen.yaml")

	if err := writeDefaultConfig(path, false); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(data) != defaultConfigTemplate() {
		t.Fatalf("unexpected config content:\n%s", string(data))
	}
}

func TestWriteDefaultConfigRejectsExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schemagen.yaml")
	if err := os.WriteFile(path, []byte("dsn: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeDefaultConfig(path, false)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing file error, got %v", err)
	}
}

func TestWriteDefaultConfigOverwritesWithForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schemagen.yaml")
	if err := os.WriteFile(path, []byte("dsn: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeDefaultConfig(path, true); err != nil {
		t.Fatalf("force overwrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(data) != defaultConfigTemplate() {
		t.Fatalf("unexpected config content after overwrite:\n%s", string(data))
	}
}
