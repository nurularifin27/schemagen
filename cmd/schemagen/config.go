package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nurularifin27/schemagen/dbtype"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	DSN          string   `yaml:"dsn"`
	Driver       string   `yaml:"driver"`
	OutDir       string   `yaml:"out_dir"`
	Tables       []string `yaml:"tables"`
	Exclude      []string `yaml:"exclude"`
	OnConflict   string   `yaml:"on_conflict"`
	TypeStrategy string   `yaml:"type_strategy"`
}

const (
	defaultDriver       = "postgres"
	defaultConfig       = "schemagen.yaml"
	defaultOutDir       = "./internal/entity"
	defaultOnConflict   = "skip"
	defaultTypeStrategy = dbtype.StrategyDriver
)

var defaultExclude = []string{"schema_migrations", "goose_db_version", "migrations"}

func defaultConfigTemplate() string {
	return strings.TrimSpace(fmt.Sprintf(`
# Database connection string.
dsn: ""

# Supported drivers: postgres, mysql, mariadb, sqlite
driver: %s

# Type strategy:
# - driver: preserve driver-aware types where possible (uuid.UUID, decimal.Decimal, datatypes.Date, datatypes.Time, pgtype arrays)
# - gorm: prefer simpler GORM-friendly scalar types (string, float64, time.Time) while arrays remain driver-aware
type_strategy: %s

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
`, defaultDriver, defaultTypeStrategy, defaultOutDir, defaultExclude[0], defaultExclude[1], defaultExclude[2], defaultOnConflict)) + "\n"
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
	if len(cfg.Exclude) == 0 {
		cfg.Exclude = append([]string(nil), defaultExclude...)
	}
	if cfg.OnConflict == "" {
		cfg.OnConflict = defaultOnConflict
	}
	if cfg.TypeStrategy == "" {
		cfg.TypeStrategy = defaultTypeStrategy
	}
	cfg.Driver = strings.ToLower(strings.TrimSpace(cfg.Driver))
	cfg.OnConflict = strings.ToLower(strings.TrimSpace(cfg.OnConflict))
	cfg.TypeStrategy = strings.ToLower(strings.TrimSpace(cfg.TypeStrategy))
}

func isValidConflictPolicy(policy string) bool {
	switch policy {
	case "skip", "error", "backup", "overwrite":
		return true
	default:
		return false
	}
}

func isValidTypeStrategy(strategy string) bool {
	switch strategy {
	case dbtype.StrategyDriver, dbtype.StrategyGorm:
		return true
	default:
		return false
	}
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
