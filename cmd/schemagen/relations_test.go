package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRelationsIfExistsReadsYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations.yaml")
	content := []byte("relations:\n  - table: orders\n    kind: belongs_to\n    field: User\n    target_table: users\n    foreign_key: user_id\n    target_key: id\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := loadRelationsIfExists(path)
	if len(cfg.Relations) != 1 || cfg.Relations[0].Field != "User" {
		t.Fatalf("unexpected relations config: %#v", cfg)
	}
}

func TestValidateRelationsConfig(t *testing.T) {
	cfg := RelationsConfig{
		Relations: []RelationConfig{{
			Table:       "orders",
			Kind:        "belongs_to",
			TargetTable: "users",
			ForeignKey:  "user_id",
			TargetKey:   "id",
		}},
	}
	if err := validateRelationsConfig(cfg); err != nil {
		t.Fatalf("expected config to be valid, got %v", err)
	}

	bad := RelationsConfig{
		Relations: []RelationConfig{{
			Table:       "users",
			Kind:        "many_to_many",
			Field:       "Roles",
			TargetTable: "roles",
		}},
	}
	if err := validateRelationsConfig(bad); err == nil {
		t.Fatal("expected invalid many_to_many config to fail")
	}
}

func TestNormalizeRelationsConfigFillsDefaultField(t *testing.T) {
	cfg := RelationsConfig{
		Relations: []RelationConfig{
			{
				Table:       "orders",
				Kind:        "belongs_to",
				TargetTable: "users",
				ForeignKey:  "user_id",
				TargetKey:   "id",
			},
			{
				Table:       "users",
				Kind:        "has_many",
				TargetTable: "order_items",
				ForeignKey:  "user_id",
				TargetKey:   "id",
			},
		},
	}

	normalizeRelationsConfig(&cfg)
	if cfg.Relations[0].Field != "User" {
		t.Fatalf("expected belongs_to field User, got %q", cfg.Relations[0].Field)
	}
	if cfg.Relations[1].Field != "OrderItems" {
		t.Fatalf("expected has_many field OrderItems, got %q", cfg.Relations[1].Field)
	}
}

func TestWriteDefaultRelationsConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations.yaml")
	if err := writeDefaultRelationsConfig(path, false); err != nil {
		t.Fatalf("write default relations config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read relations config: %v", err)
	}
	if string(data) != defaultRelationsTemplate() {
		t.Fatalf("unexpected relations config content:\n%s", string(data))
	}
}

func TestWriteDefaultRelationsConfigRejectsExistingWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations.yaml")
	if err := os.WriteFile(path, []byte("relations: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeDefaultRelationsConfig(path, false)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing relations config error, got %v", err)
	}
}
