package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	DSN              string               `yaml:"dsn"`
	Driver           string               `yaml:"driver"`
	Renderer         string               `yaml:"renderer"`
	OutDir           string               `yaml:"out_dir"`
	Tables           []string             `yaml:"tables"`
	Exclude          []string             `yaml:"exclude"`
	OnConflict       string               `yaml:"on_conflict"`
	DecimalStrategy  string               `yaml:"decimal_strategy"`
	JSONStrategy     string               `yaml:"json_strategy"`
	NullableStrategy string               `yaml:"nullable_strategy"`
	TypeOverrides    []TypeOverrideConfig `yaml:"type_overrides"`
}

type TypeOverrideConfig struct {
	Table   string   `yaml:"table"`
	Column  string   `yaml:"column"`
	DBType  string   `yaml:"db_type"`
	GoType  string   `yaml:"go_type"`
	Imports []string `yaml:"imports"`
}

const (
	defaultDriver           = "postgres"
	defaultRenderer         = "sqlx"
	defaultConfig           = "schemagen.yaml"
	defaultOutDir           = "./internal/entity"
	defaultOnConflict       = "skip"
	defaultDecimalStrategy  = "float64"
	defaultJSONStrategy     = "bytes"
	defaultNullableStrategy = "pointer"
)

var defaultExclude = []string{"schema_migrations", "goose_db_version", "migrations"}

func defaultConfigTemplate() string {
	return strings.TrimSpace(fmt.Sprintf(`
# Database connection string.
dsn: ""

# Supported drivers: postgres, mysql, mariadb, sqlite
driver: %s

# Output renderer: plain, sqlx, gorm
renderer: %s

# Output directory for generated entities.
out_dir: %s

# Optional include list. Leave empty to inspect all tables.
tables: []

# Tables to ignore during generation.
exclude:
  - %s
  - %s
  - %s

# Conflict policy for existing unmanaged files:
# - skip: leave file as-is and print a warning
# - error: fail immediately
# - backup: rename existing file to *.bak.<timestamp>, then write generated file
# - overwrite: replace existing file
on_conflict: %s

# Decimal mapping strategy: float64, string
decimal_strategy: %s

# JSON mapping strategy: bytes, rawmessage
json_strategy: %s

# Nullable mapping strategy: pointer, sqlnull
nullable_strategy: %s

# Optional explicit type overrides.
type_overrides: []
`, defaultDriver, defaultRenderer, defaultOutDir, defaultExclude[0], defaultExclude[1], defaultExclude[2], defaultOnConflict, defaultDecimalStrategy, defaultJSONStrategy, defaultNullableStrategy)) + "\n"
}

func loadConfigIfExists(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}
		}
		log.Fatalf("failed to read config %q: %v", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("invalid YAML in %q: %v", path, err)
	}
	return cfg
}

func normalizeConfig(cfg *Config) {
	if cfg.Driver == "" {
		cfg.Driver = defaultDriver
	}
	if cfg.OutDir == "" {
		cfg.OutDir = defaultOutDir
	}
	if cfg.Renderer == "" {
		cfg.Renderer = defaultRenderer
	}
	if len(cfg.Exclude) == 0 {
		cfg.Exclude = append([]string(nil), defaultExclude...)
	}
	if cfg.OnConflict == "" {
		cfg.OnConflict = defaultOnConflict
	}
	if cfg.DecimalStrategy == "" {
		cfg.DecimalStrategy = defaultDecimalStrategy
	}
	if cfg.JSONStrategy == "" {
		cfg.JSONStrategy = defaultJSONStrategy
	}
	if cfg.NullableStrategy == "" {
		cfg.NullableStrategy = defaultNullableStrategy
	}
	cfg.Driver = strings.ToLower(strings.TrimSpace(cfg.Driver))
	cfg.Renderer = strings.ToLower(strings.TrimSpace(cfg.Renderer))
	cfg.OnConflict = strings.ToLower(strings.TrimSpace(cfg.OnConflict))
	cfg.DecimalStrategy = strings.ToLower(strings.TrimSpace(cfg.DecimalStrategy))
	cfg.JSONStrategy = strings.ToLower(strings.TrimSpace(cfg.JSONStrategy))
	cfg.NullableStrategy = strings.ToLower(strings.TrimSpace(cfg.NullableStrategy))
}

func isValidConflictPolicy(policy string) bool {
	switch policy {
	case "skip", "error", "backup", "overwrite":
		return true
	default:
		return false
	}
}

func isValidRenderer(renderer string) bool {
	switch renderer {
	case "plain", "sqlx", "gorm":
		return true
	default:
		return false
	}
}

func isValidDecimalStrategy(strategy string) bool {
	switch strategy {
	case "float64", "string":
		return true
	default:
		return false
	}
}

func isValidJSONStrategy(strategy string) bool {
	switch strategy {
	case "bytes", "rawmessage":
		return true
	default:
		return false
	}
}

func isValidNullableStrategy(strategy string) bool {
	switch strategy {
	case "pointer", "sqlnull":
		return true
	default:
		return false
	}
}

func normalizeTypeOverrides(overrides []TypeOverrideConfig) []TypeOverrideConfig {
	if len(overrides) == 0 {
		return nil
	}

	normalized := make([]TypeOverrideConfig, 0, len(overrides))
	for _, override := range overrides {
		override.Table = strings.ToLower(strings.TrimSpace(override.Table))
		override.Column = strings.ToLower(strings.TrimSpace(override.Column))
		override.DBType = strings.ToLower(strings.TrimSpace(override.DBType))
		override.GoType = strings.TrimSpace(override.GoType)
		for i := range override.Imports {
			override.Imports[i] = strings.TrimSpace(override.Imports[i])
		}
		normalized = append(normalized, override)
	}
	return normalized
}

func validateTypeOverrides(overrides []TypeOverrideConfig) error {
	for _, override := range overrides {
		if override.GoType == "" {
			return fmt.Errorf("type_overrides.go_type is required")
		}
		if override.Table == "" && override.Column == "" && override.DBType == "" {
			return fmt.Errorf("type_overrides entry must set at least one of table, column, or db_type")
		}
	}
	return nil
}

func writeDefaultConfig(path string, force bool) error {
	if strings.TrimSpace(path) == "" {
		path = defaultConfig
	}

	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config %q already exists; rerun with --force to overwrite", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat config %q: %w", path, err)
		}
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config directory %q: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, []byte(defaultConfigTemplate()), 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}

	return nil
}

func connectDB(driver, dsn string) *gorm.DB {
	var dialector gorm.Dialector
	switch strings.ToLower(driver) {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql", "mariadb":
		dialector = mysql.Open(dsn)
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(dsn)
	default:
		log.Fatalf("unsupported driver %q (supported: postgres, mysql, mariadb, sqlite)", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	return db
}
