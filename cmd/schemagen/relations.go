package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nurularifin27/schemagen/entitygen"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/schema"
)

const (
	defaultRelationsConfig     = "schemagen.relations"
	legacyRelationsConfig      = "schemagen.relations.yaml"
	defaultRelationsConfigFile = "default.yaml"
)

type RelationsConfig struct {
	Relations []RelationConfig                `yaml:"relations"`
	Tables    map[string]TableRelationsConfig `yaml:"tables"`
}

type TableRelationsConfig struct {
	Relations []RelationConfig `yaml:"relations"`
}

type RelationConfig struct {
	Table          string `yaml:"table"`
	Kind           string `yaml:"kind"`
	Field          string `yaml:"field"`
	TargetTable    string `yaml:"target_table"`
	ForeignKey     string `yaml:"foreign_key"`
	TargetKey      string `yaml:"target_key"`
	JoinTable      string `yaml:"join_table"`
	JoinForeignKey string `yaml:"join_foreign_key"`
	JoinTargetKey  string `yaml:"join_target_key"`
	SourceKey      string `yaml:"source_key"`
}

func loadRelationsIfExists(path string) (RelationsConfig, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultRelationsConfig
	}
	if path == defaultRelationsConfig {
		return loadDefaultRelationsIfExists()
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RelationsConfig{}, nil
		}
		return RelationsConfig{}, fmt.Errorf("failed to read relations config %q: %w", path, err)
	}
	if info.IsDir() {
		return loadRelationsDir(path)
	}
	return loadRelationsFile(path)
}

func loadDefaultRelationsIfExists() (RelationsConfig, error) {
	if info, err := os.Stat(defaultRelationsConfig); err == nil {
		if info.IsDir() {
			return loadRelationsDir(defaultRelationsConfig)
		}
		return loadRelationsFile(defaultRelationsConfig)
	} else if err != nil && !os.IsNotExist(err) {
		return RelationsConfig{}, fmt.Errorf("failed to read relations config %q: %w", defaultRelationsConfig, err)
	}

	if info, err := os.Stat(legacyRelationsConfig); err == nil && !info.IsDir() {
		return loadRelationsFile(legacyRelationsConfig)
	} else if err != nil && !os.IsNotExist(err) {
		return RelationsConfig{}, fmt.Errorf("failed to read legacy relations config %q: %w", legacyRelationsConfig, err)
	}

	return RelationsConfig{}, nil
}

func loadRelationsFile(path string) (RelationsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RelationsConfig{}, fmt.Errorf("failed to read relations config %q: %w", path, err)
	}
	var cfg RelationsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return RelationsConfig{}, fmt.Errorf("invalid YAML in %q: %w", path, err)
	}
	cfg.flattenGroupedRelations()
	return cfg, nil
}

func loadRelationsDir(path string) (RelationsConfig, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return RelationsConfig{}, fmt.Errorf("failed to read relations config directory %q: %w", path, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	merged := RelationsConfig{}
	for _, name := range names {
		cfg, err := loadRelationsFile(filepath.Join(path, name))
		if err != nil {
			return RelationsConfig{}, err
		}
		merged.Relations = append(merged.Relations, cfg.Relations...)
	}
	return merged, nil
}

func (cfg *RelationsConfig) flattenGroupedRelations() {
	if len(cfg.Tables) == 0 {
		return
	}

	flattened := make([]RelationConfig, 0, len(cfg.Relations))
	flattened = append(flattened, cfg.Relations...)
	for tableName, tableCfg := range cfg.Tables {
		for _, rel := range tableCfg.Relations {
			if strings.TrimSpace(rel.Table) == "" {
				rel.Table = tableName
			}
			flattened = append(flattened, rel)
		}
	}
	cfg.Relations = flattened
}

func normalizeRelationsConfig(cfg *RelationsConfig) {
	for i := range cfg.Relations {
		cfg.Relations[i].Table = strings.ToLower(strings.TrimSpace(cfg.Relations[i].Table))
		cfg.Relations[i].Kind = strings.ToLower(strings.TrimSpace(cfg.Relations[i].Kind))
		cfg.Relations[i].Field = strings.TrimSpace(cfg.Relations[i].Field)
		cfg.Relations[i].TargetTable = strings.ToLower(strings.TrimSpace(cfg.Relations[i].TargetTable))
		cfg.Relations[i].ForeignKey = strings.TrimSpace(cfg.Relations[i].ForeignKey)
		cfg.Relations[i].TargetKey = strings.TrimSpace(cfg.Relations[i].TargetKey)
		cfg.Relations[i].JoinTable = strings.ToLower(strings.TrimSpace(cfg.Relations[i].JoinTable))
		cfg.Relations[i].JoinForeignKey = strings.TrimSpace(cfg.Relations[i].JoinForeignKey)
		cfg.Relations[i].JoinTargetKey = strings.TrimSpace(cfg.Relations[i].JoinTargetKey)
		cfg.Relations[i].SourceKey = strings.TrimSpace(cfg.Relations[i].SourceKey)
		if cfg.Relations[i].Field == "" && cfg.Relations[i].TargetTable != "" {
			cfg.Relations[i].Field = defaultRelationField(cfg.Relations[i].Kind, cfg.Relations[i].TargetTable)
		}
	}
}

func validateRelationsConfig(cfg RelationsConfig) error {
	seen := make(map[string]struct{}, len(cfg.Relations))
	for _, rel := range cfg.Relations {
		if rel.Table == "" {
			return fmt.Errorf("relations.table is required")
		}
		if rel.TargetTable == "" {
			return fmt.Errorf("relations.target_table is required")
		}
		switch rel.Kind {
		case "belongs_to", "has_one", "has_many":
			if rel.ForeignKey == "" {
				return fmt.Errorf("relations.foreign_key is required for %s", rel.Kind)
			}
			if rel.TargetKey == "" {
				return fmt.Errorf("relations.target_key is required for %s", rel.Kind)
			}
		case "many_to_many":
			if rel.JoinTable == "" || rel.JoinForeignKey == "" || rel.JoinTargetKey == "" {
				return fmt.Errorf("relations join_table, join_foreign_key, and join_target_key are required for many_to_many")
			}
			if rel.TargetKey == "" {
				return fmt.Errorf("relations.target_key is required for many_to_many")
			}
			if rel.SourceKey == "" {
				return fmt.Errorf("relations.source_key is required for many_to_many")
			}
		default:
			return fmt.Errorf("unsupported relation kind %q", rel.Kind)
		}

		key := relationSignature(rel)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate relation definition for table=%q kind=%q field=%q target_table=%q", rel.Table, rel.Kind, rel.Field, rel.TargetTable)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func relationSignature(rel RelationConfig) string {
	return strings.Join([]string{
		rel.Table,
		rel.Kind,
		rel.Field,
		rel.TargetTable,
		rel.ForeignKey,
		rel.TargetKey,
		rel.JoinTable,
		rel.JoinForeignKey,
		rel.JoinTargetKey,
		rel.SourceKey,
	}, "|")
}

func defaultRelationField(kind, targetTable string) string {
	base := schema.NamingStrategy{}.SchemaName(targetTable)
	switch kind {
	case "has_many", "many_to_many":
		return pluralizeFieldName(base)
	default:
		return base
	}
}

func pluralizeFieldName(name string) string {
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "y") && len(name) > 1 {
		prev := lower[len(lower)-2]
		if !strings.ContainsRune("aeiou", rune(prev)) {
			return name[:len(name)-1] + "ies"
		}
	}
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "z") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		return name + "es"
	}
	return name + "s"
}

func toEntityRelations(cfg RelationsConfig) []entitygen.Relation {
	if len(cfg.Relations) == 0 {
		return nil
	}

	result := make([]entitygen.Relation, 0, len(cfg.Relations))
	for _, rel := range cfg.Relations {
		result = append(result, entitygen.Relation{
			Table:          rel.Table,
			Kind:           rel.Kind,
			Field:          rel.Field,
			TargetTable:    rel.TargetTable,
			ForeignKey:     rel.ForeignKey,
			TargetKey:      rel.TargetKey,
			JoinTable:      rel.JoinTable,
			JoinForeignKey: rel.JoinForeignKey,
			JoinTargetKey:  rel.JoinTargetKey,
			SourceKey:      rel.SourceKey,
		})
	}
	return result
}

func defaultRelationsTemplate() string {
	return strings.TrimSpace(`
# Explicit relation mapping. Leave empty if you do not want generated relation fields.
tables: {}

# Recommended grouped format. Split files by domain if your schema is large:
# schemagen.relations/
#   auth.yaml
#   org.yaml
#   catalog.yaml
#   inventory.yaml
#   sales.yaml
#   menu.yaml
#
# Example file content:
# tables:
#   orders:
#     relations:
#       - kind: belongs_to
#         target_table: users
#         foreign_key: user_id
#         target_key: id
#
#   users:
#     relations:
#       - kind: has_many
#         target_table: orders
#         foreign_key: user_id
#         target_key: id
#
#       - kind: has_one
#         target_table: user_profiles
#         foreign_key: user_id
#         target_key: id
#
#       - kind: many_to_many
#         target_table: roles
#         join_table: user_roles
#         join_foreign_key: user_id
#         join_target_key: role_id
#         source_key: id
#         target_key: id
#
# Legacy flat format is still supported:
# relations:
#   - table: orders
#     kind: belongs_to
#     target_table: users
#     foreign_key: user_id
#     target_key: id
`) + "\n"
}

func writeDefaultRelationsConfig(path string, force bool) error {
	if strings.TrimSpace(path) == "" {
		path = defaultRelationsConfig
	}
	path = strings.TrimSpace(path)
	if strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml") {
		return writeDefaultRelationsFile(path, force)
	}
	return writeDefaultRelationsDir(path, force)
}

func writeDefaultRelationsFile(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("relations config %q already exists; rerun with --force to overwrite", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat relations config %q: %w", path, err)
		}
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create relations config directory %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(defaultRelationsTemplate()), 0o644); err != nil {
		return fmt.Errorf("write relations config %q: %w", path, err)
	}
	return nil
}

func writeDefaultRelationsDir(path string, force bool) error {
	target := filepath.Join(path, defaultRelationsConfigFile)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return fmt.Errorf("relations config path %q exists and is not a directory", path)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat relations config directory %q: %w", path, err)
	}
	if !force {
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("relations config %q already exists; rerun with --force to overwrite", target)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat relations config %q: %w", target, err)
		}
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create relations config directory %q: %w", path, err)
	}
	if err := os.WriteFile(target, []byte(defaultRelationsTemplate()), 0o644); err != nil {
		return fmt.Errorf("write relations config %q: %w", target, err)
	}
	return nil
}
