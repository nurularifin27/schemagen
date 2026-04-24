package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfigIfExistsMissingFile(t *testing.T) {
	cfg := loadConfigIfExists(filepath.Join(t.TempDir(), "missing.yaml"))
	if !reflect.DeepEqual(cfg, Config{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadConfigIfExistsReadsYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schemagen.yaml")
	content := []byte("dsn: postgres://x\n" +
		"driver: postgres\n" +
		"type_strategy: gorm\n" +
		"out_dir: ./entity\n" +
		"tables:\n  - users\n" +
		"exclude:\n  - migrations\n" +
		"on_conflict: backup\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfigIfExists(path)
	if cfg.DSN != "postgres://x" || cfg.Driver != "postgres" || cfg.TypeStrategy != "gorm" || cfg.OutDir != "./entity" || cfg.OnConflict != "backup" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.Tables, []string{"users"}) {
		t.Fatalf("unexpected tables: %#v", cfg.Tables)
	}
	if !reflect.DeepEqual(cfg.Exclude, []string{"migrations"}) {
		t.Fatalf("unexpected exclude: %#v", cfg.Exclude)
	}
}

func TestNormalizeConfigAppliesDefaults(t *testing.T) {
	cfg := Config{}
	normalizeConfig(&cfg)

	if cfg.Driver != defaultDriver {
		t.Fatalf("expected driver %q, got %q", defaultDriver, cfg.Driver)
	}
	if cfg.TypeStrategy != defaultTypeStrategy {
		t.Fatalf("expected type strategy %q, got %q", defaultTypeStrategy, cfg.TypeStrategy)
	}
	if cfg.OutDir != defaultOutDir {
		t.Fatalf("expected out dir %q, got %q", defaultOutDir, cfg.OutDir)
	}
	if cfg.OnConflict != defaultOnConflict {
		t.Fatalf("expected on conflict %q, got %q", defaultOnConflict, cfg.OnConflict)
	}
	if !reflect.DeepEqual(cfg.Exclude, defaultExclude) {
		t.Fatalf("expected default exclude %#v, got %#v", defaultExclude, cfg.Exclude)
	}
}

func TestNormalizeConfigNormalizesCase(t *testing.T) {
	cfg := Config{
		Driver:       "Postgres",
		TypeStrategy: "GORM",
		OnConflict:   "BACKUP",
	}
	normalizeConfig(&cfg)

	if cfg.Driver != "postgres" {
		t.Fatalf("expected normalized driver, got %q", cfg.Driver)
	}
	if cfg.TypeStrategy != "gorm" {
		t.Fatalf("expected normalized type strategy, got %q", cfg.TypeStrategy)
	}
	if cfg.OnConflict != "backup" {
		t.Fatalf("expected normalized on conflict, got %q", cfg.OnConflict)
	}
}

func TestMergeConfigPrefersOverride(t *testing.T) {
	base := Config{
		DSN:          "dsn-base",
		Driver:       "postgres",
		TypeStrategy: "driver",
		OutDir:       "./base",
		Tables:       []string{"users"},
		Exclude:      []string{"migrations"},
		OnConflict:   "skip",
	}
	override := Config{
		DSN:          "dsn-override",
		TypeStrategy: "gorm",
		OutDir:       "./override",
		Tables:       []string{"companies"},
		OnConflict:   "backup",
	}

	cfg := mergeConfig(base, override)
	if cfg.DSN != "dsn-override" || cfg.OutDir != "./override" || cfg.Driver != "postgres" || cfg.TypeStrategy != "gorm" || cfg.OnConflict != "backup" {
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

func TestIsValidTypeStrategy(t *testing.T) {
	valid := []string{"driver", "gorm"}
	for _, strategy := range valid {
		if !isValidTypeStrategy(strategy) {
			t.Fatalf("expected strategy %q to be valid", strategy)
		}
	}
	if isValidTypeStrategy("random") {
		t.Fatal("expected random strategy to be invalid")
	}
}

func TestDefaultConfigTemplateIncludesDefaults(t *testing.T) {
	content := defaultConfigTemplate()

	required := []string{
		"dsn: \"\"",
		"driver: postgres",
		"type_strategy: driver",
		"out_dir: ./internal/entity",
		"on_conflict: skip",
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
