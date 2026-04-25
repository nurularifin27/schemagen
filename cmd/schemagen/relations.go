package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nurularifin27/schemagen/entitygen"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/schema"
)

const defaultRelationsConfig = "schemagen.relations.yaml"

type RelationsConfig struct {
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

func loadRelationsIfExists(path string) RelationsConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RelationsConfig{}
		}
		log.Fatalf("failed to read relations config %q: %v", path, err)
	}

	var cfg RelationsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("invalid YAML in %q: %v", path, err)
	}
	return cfg
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
	}
	return nil
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
relations: []

# Example:
# relations:
#   - table: orders
#     kind: belongs_to
#     target_table: users
#     foreign_key: user_id
#     target_key: id
#
#   - table: users
#     kind: has_many
#     target_table: orders
#     foreign_key: user_id
#     target_key: id
#
#   - table: users
#     kind: has_one
#     target_table: user_profiles
#     foreign_key: user_id
#     target_key: id
#
#   - table: users
#     kind: many_to_many
#     target_table: roles
#     join_table: user_roles
#     join_foreign_key: user_id
#     join_target_key: role_id
#     source_key: id
#     target_key: id
`) + "\n"
}

func writeDefaultRelationsConfig(path string, force bool) error {
	if strings.TrimSpace(path) == "" {
		path = defaultRelationsConfig
	}

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
