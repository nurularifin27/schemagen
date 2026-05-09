package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRelationsIfExistsReadsYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations.yaml")
	content := []byte("tables:\n  orders:\n    relations:\n      - kind: belongs_to\n        field: User\n        target_table: users\n        foreign_key: user_id\n        target_key: id\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadRelationsIfExists(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Relations) != 1 || cfg.Relations[0].Field != "User" || cfg.Relations[0].Table != "orders" {
		t.Fatalf("unexpected relations config: %#v", cfg)
	}
}

func TestLoadRelationsIfExistsReadsLegacyFlatYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations.yaml")
	content := []byte("relations:\n  - table: orders\n    kind: belongs_to\n    field: User\n    target_table: users\n    foreign_key: user_id\n    target_key: id\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadRelationsIfExists(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Relations) != 1 || cfg.Relations[0].Field != "User" || cfg.Relations[0].Table != "orders" {
		t.Fatalf("unexpected legacy relations config: %#v", cfg)
	}
}

func TestLoadRelationsIfExistsReadsDirectoryYAML(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "schemagen.relations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	first := []byte("tables:\n  orders:\n    relations:\n      - kind: belongs_to\n        target_table: users\n        foreign_key: user_id\n        target_key: id\n")
	second := []byte("tables:\n  users:\n    relations:\n      - kind: has_many\n        target_table: orders\n        foreign_key: user_id\n        target_key: id\n")
	if err := os.WriteFile(filepath.Join(dir, "orders.yaml"), first, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "users.yaml"), second, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadRelationsIfExists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Relations) != 2 {
		t.Fatalf("expected 2 merged relations, got %#v", cfg.Relations)
	}
}

func TestLoadRelationsIfExistsDefaultFallsBackToLegacyFile(t *testing.T) {
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	content := []byte("tables:\n  orders:\n    relations:\n      - kind: belongs_to\n        target_table: users\n        foreign_key: user_id\n        target_key: id\n")
	if err := os.WriteFile(legacyRelationsConfig, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadRelationsIfExists(defaultRelationsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Relations) != 1 || cfg.Relations[0].Table != "orders" {
		t.Fatalf("unexpected fallback config: %#v", cfg)
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

func TestValidateRelationsConfigRejectsDuplicateRelations(t *testing.T) {
	cfg := RelationsConfig{
		Relations: []RelationConfig{
			{
				Table:       "orders",
				Kind:        "belongs_to",
				Field:       "User",
				TargetTable: "users",
				ForeignKey:  "user_id",
				TargetKey:   "id",
			},
			{
				Table:       "orders",
				Kind:        "belongs_to",
				Field:       "User",
				TargetTable: "users",
				ForeignKey:  "user_id",
				TargetKey:   "id",
			},
		},
	}
	if err := validateRelationsConfig(cfg); err == nil || !strings.Contains(err.Error(), "duplicate relation definition") {
		t.Fatalf("expected duplicate relation error, got %v", err)
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
			{
				Table:          "users",
				Kind:           "many_to_many",
				TargetTable:    "roles",
				JoinTable:      "user_roles",
				JoinForeignKey: "user_id",
				JoinTargetKey:  "role_id",
				SourceKey:      "id",
				TargetKey:      "id",
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
	if cfg.Relations[2].Field != "Roles" {
		t.Fatalf("expected many_to_many field Roles, got %q", cfg.Relations[2].Field)
	}
}

func TestWriteDefaultRelationsConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations")
	if err := writeDefaultRelationsConfig(path, false); err != nil {
		t.Fatalf("write default relations config: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(path, defaultRelationsConfigFile))
	if err != nil {
		t.Fatalf("read relations config: %v", err)
	}
	if string(data) != defaultRelationsTemplate() {
		t.Fatalf("unexpected relations config content:\n%s", string(data))
	}
}

func TestWriteDefaultRelationsConfigRejectsExistingWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.relations")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, defaultRelationsConfigFile), []byte("relations: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeDefaultRelationsConfig(path, false)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing relations config error, got %v", err)
	}
}
